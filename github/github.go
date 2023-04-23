package github

import (
	"context"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/google/go-github/v41/github"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Repository struct {
	Name  string
	Owner struct {
		Login string
	}
}

type User struct {
	Login                     string
	RepositoriesContributedTo struct {
		Nodes    []Repository
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"repositoriesContributedTo(contributionTypes: [COMMIT, PULL_REQUEST, REPOSITORY], first: 100, after: $reposCursor)"`
}

type Query struct {
	User User `graphql:"user(login: $userLogin)"`
}

type V4Repo struct {
	User       string
	Repository string
}

func getv4(token, user string) []V4Repo {
	repos := []V4Repo{}
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauth2Client := oauth2.NewClient(context.Background(), tokenSource)
	client := githubv4.NewClient(oauth2Client)

	var query Query
	variables := map[string]interface{}{
		"userLogin":   githubv4.String(user), // Replace with the username you want to retrieve contributed projects from
		"reposCursor": (*githubv4.String)(nil),
	}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			log.Error().
				Str("stage", "github").
				Msg(err.Error())
			return []V4Repo{}
		}

		projects := query.User.RepositoriesContributedTo.Nodes
		for _, project := range projects {
			repos = append(repos, V4Repo{User: project.Owner.Login, Repository: project.Name})
		}

		if !query.User.RepositoriesContributedTo.PageInfo.HasNextPage {
			break
		}
		variables["reposCursor"] = githubv4.NewString(query.User.RepositoriesContributedTo.PageInfo.EndCursor)
	}
	return repos
}

func addWiki(r github.Repository, repo types.GenRepo, token string) types.Repo {
	if !(r.GetHasWiki() && repo.Wiki &&
		types.StatRemote(r.GetCloneURL(), r.GetSSHURL(), repo)) {
		return types.Repo{}
	}

	return types.Repo{
		Name:          *r.Name + ".wiki",
		URL:           types.DotGitRx.ReplaceAllString(r.GetCloneURL(), ".wiki.git"),
		SSHURL:        types.DotGitRx.ReplaceAllString(r.GetSSHURL(), ".wiki.git"),
		Token:         token,
		Defaultbranch: r.GetDefaultBranch(),
		Origin:        repo,
		Owner:         r.GetOwner().GetLogin(),
		Hoster:        "github.com",
		Description:   r.GetDescription(),
		Private:       r.GetPrivate(),
	}
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Github {
		err := repo.Filter.ParseDuration()
		if err != nil {
			log.Error().
				Str("stage", "github").
				Str("url", repo.URL).
				Msg(err.Error())
		}
		ran = true
		if repo.User == "" {
			log.Info().
				Str("stage", "github").
				Str("url", "https://github.com").
				Msg("grabbing my repositories")
		} else {
			log.Info().
				Str("stage", "github").
				Str("url", "https://github.com").
				Msgf("grabbing the repositories from %s", repo.User)
		}

		opt := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				PerPage: 50,
			},
		}

		i := 1
		githubrepos := []*github.Repository{}
		token := repo.GetToken()

		var client *github.Client
		if token == "" {
			client = github.NewClient(nil)
		} else {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			tc := oauth2.NewClient(context.TODO(), ts)

			client = github.NewClient(tc)
		}

		v4user := repo.User
		if token != "" {
			user, _, err := client.Users.Get(context.TODO(), "")
			if err != nil {
				log.Error().
					Str("stage", "github").
					Str("url", "https://github.com").
					Msg(err.Error())
				continue
			}

			if repo.User == user.GetLogin() {
				repo.User = ""
				v4user = user.GetLogin()
			}
		}

		if token != "" && v4user != "" && repo.Contributed {
			for _, r := range getv4(token, v4user) {
				github_repo, _, err := client.Repositories.Get(context.Background(), r.User, r.Repository)
				if err != nil {
					log.Error().
						Str("stage", "github").
						Str("url", "https://github.com").
						Msg(err.Error())
					continue
				}
				githubrepos = append(githubrepos, github_repo)
			}
		}

		for {
			opt.Page = i
			repos, _, err := client.Repositories.List(context.TODO(), repo.User, opt)
			if err != nil {
				log.Error().
					Str("stage", "github").
					Str("url", "https://github.com").
					Msg(err.Error())
				continue
			}

			if len(repos) == 0 {
				break
			}
			githubrepos = append(githubrepos, repos...)
			i++
		}

		if repo.Starred {
			i = 1
			opt := &github.ActivityListStarredOptions{
				ListOptions: github.ListOptions{
					PerPage: 50,
				},
			}

			for {
				opt.ListOptions.Page = i
				repos, _, err := client.Activity.ListStarred(context.TODO(), repo.User, opt)
				if err != nil {
					log.Error().
						Str("stage", "github").
						Str("url", "https://github.com").
						Msg(err.Error())
				}
				if len(repos) == 0 {
					break
				}
				for _, starredrepo := range repos {
					githubrepos = append(githubrepos, starredrepo.Repository)
				}
				i++
			}
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)
		for i := range repo.Filter.Languages {
			repo.Filter.Languages[i] = strings.ToLower(repo.Filter.Languages[i])
		}
		languages := types.GetMap(repo.Filter.Languages)

		for _, r := range githubrepos {
			if repo.Filter.ExcludeArchived {
				if r.Archived != nil {
					if *r.Archived {
						continue
					}
				}
			}
			if len(repo.Filter.Languages) > 0 {
				if r.Language != nil {
					if !languages[strings.ToLower(*r.Language)] {
						continue
					}
				}
			}
			if *r.StargazersCount < repo.Filter.Stars {
				continue
			}
			if time.Since(r.PushedAt.Time) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
				continue
			}

			if include[*r.Name] {
				repos = append(repos, types.Repo{
					Name:          r.GetName(),
					URL:           r.GetCloneURL(),
					SSHURL:        r.GetSSHURL(),
					Token:         token,
					Defaultbranch: r.GetDefaultBranch(),
					Origin:        repo,
					Owner:         r.GetOwner().GetLogin(),
					Hoster:        "github.com",
					Description:   r.GetDescription(),
					Private:       r.GetPrivate(),
				})
				wiki := addWiki(*r, repo, token)
				if wiki.Name != "" {
					repos = append(repos, wiki)
				}

				continue
			}

			if exclude[*r.Name] {
				continue
			}

			if excludeorgs[r.GetOwner().GetLogin()] {
				continue
			}

			if len(repo.Include) == 0 {
				if len(includeorgs) > 0 {
					if includeorgs[r.GetOwner().GetLogin()] {
						repos = append(repos, types.Repo{
							Name:          r.GetName(),
							URL:           r.GetCloneURL(),
							SSHURL:        r.GetSSHURL(),
							Token:         token,
							Defaultbranch: r.GetDefaultBranch(),
							Origin:        repo,
							Owner:         r.GetOwner().GetLogin(),
							Hoster:        "github.com",
							Description:   r.GetDescription(),
							Private:       r.GetPrivate(),
						})
						wiki := addWiki(*r, repo, token)
						if wiki.Name != "" {
							repos = append(repos, wiki)
						}
					}
				} else {
					repos = append(repos, types.Repo{
						Name:          r.GetName(),
						URL:           r.GetCloneURL(),
						SSHURL:        r.GetSSHURL(),
						Token:         token,
						Defaultbranch: r.GetDefaultBranch(),
						Origin:        repo,
						Owner:         r.GetOwner().GetLogin(),
						Hoster:        "github.com",
						Description:   r.GetDescription(),
						Private:       r.GetPrivate(),
					})
					wiki := addWiki(*r, repo, token)
					if wiki.Name != "" {
						repos = append(repos, wiki)
					}
				}
			}
		}
	}

	return repos, ran
}

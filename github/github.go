package github

import (
	"context"

	"github.com/cooperspencer/gickup/types"
	"github.com/google/go-github/v41/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

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
	}
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Github {
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

		if token != "" {
			user, _, err := client.Users.Get(context.TODO(), "")
			if err != nil {
				log.Fatal().
					Str("stage", "github").
					Str("url", "https://github.com").
					Msg(err.Error())
			}

			if repo.User == user.GetLogin() {
				repo.User = ""
			}
		}

		for {
			opt.Page = i
			repos, _, err := client.Repositories.List(context.TODO(), repo.User, opt)
			if err != nil {
				log.Fatal().
					Str("stage", "github").
					Str("url", "https://github.com").
					Msg(err.Error())
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
					log.Fatal().
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

		for _, r := range githubrepos {
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

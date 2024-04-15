package onedev

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/cooperspencer/onedev"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
)

func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}

	for _, repo := range conf.Source.OneDev {
		ran = true
		if repo.URL == "" {
			repo.URL = "https://code.onedev.io/"
		}
		sub = logger.CreateSubLogger("stage", "onedev", "url", repo.URL)
		err := repo.Filter.ParseDuration()
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}
		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		sub.Info().
			Msgf("grabbing repositories from %s", repo.User)

		if repo.Password == "" && repo.Token != "" {
			repo.Password = repo.Token
		}

		client := &onedev.Client{}

		if repo.Token != "" || repo.TokenFile != "" {
			client = onedev.NewClient(repo.URL, onedev.SetToken(repo.GetToken()))
		} else {
			if repo.Password != "" {
				client = onedev.NewClient(repo.URL, onedev.SetBasicAuth(repo.Username, repo.Password))
			} else {
				client = onedev.NewClient(repo.URL)
			}
		}

		query := onedev.ProjectQueryOptions{
			Query:  "",
			Offset: 0,
			Count:  100,
		}

		user := onedev.User{}

		if repo.User == "" {
			u, _, err := client.GetMe()
			if err != nil {
				sub.Error().
					Msg("can't find user")
				break
			}
			user = u
			repo.User = user.Name
		}

		if repo.User != "" {
			query.Query = fmt.Sprintf("owned by \"%s\"", repo.User)
		}

		userrepos, _, err := client.GetProjects(&query)
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}

		for _, r := range userrepos {
			urls, _, err := client.GetCloneUrl(r.ID)
			if err != nil {
				sub.Error().
					Msg("couldn't get clone urls")
				continue
			}
			sub.Debug().Msg(urls.HTTP)
			if repo.Filter.ExcludeForks {
				if r.ForkedFromID != 0 {
					continue
				}
			}
			if len(repo.Include) > 0 {
				if !include[r.Name] {
					continue
				}
				if exclude[r.Name] {
					continue
				}
			}

			defaultbranch, _, err := client.GetDefaultBranch(r.ID)
			if err != nil {
				sub.Error().
					Msgf("couldn't get default branch for %s", r.Name)
				defaultbranch = "main"
			}

			options := onedev.CommitQueryOptions{Query: fmt.Sprintf("branch(%s)", defaultbranch)}
			commits, _, err := client.GetCommits(r.ID, &options)
			if len(commits) > 0 {
				commit, _, err := client.GetCommit(r.ID, commits[0])
				if err != nil {
					sub.Error().
						Msgf("can't get latest commit for %s", defaultbranch)
				} else {
					lastactive := time.UnixMicro(commit.Author.When)
					if time.Since(lastactive) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
						continue
					}
				}
			}

			repos = append(repos, types.Repo{
				Name:          r.Name,
				URL:           urls.HTTP,
				SSHURL:        urls.SSH,
				Token:         repo.Token,
				Defaultbranch: defaultbranch,
				Origin:        repo,
				Owner:         repo.User,
				Hoster:        types.GetHost(repo.URL),
				Description:   r.Description,
				Issues:        GetIssues(&r, client, repo, urls.HTTP),
			})
		}

		if repo.Username != "" && repo.Password != "" && len(repo.IncludeOrgs) == 0 && user.Name != "" {
			memberships, _, err := client.GetUserMemberships(user.ID)
			if err != nil {
				sub.Error().
					Msgf("couldn't get memberships for %s", user.Name)
			}

			for _, membership := range memberships {
				group, _, err := client.GetGroup(membership.GroupID)
				if err != nil {
					sub.Error().
						Msgf("couldn't get group with id %d", membership.GroupID)
				}
				if !excludeorgs[group.Name] {
					repo.IncludeOrgs = append(repo.IncludeOrgs, group.Name)
				}
			}
		}

		if len(repo.IncludeOrgs) > 0 {
			for _, org := range repo.IncludeOrgs {
				query.Query = fmt.Sprintf("children of \"%s\"", org)

				orgrepos, _, err := client.GetProjects(&query)
				if err != nil {
					sub.Error().
						Msg(err.Error())
				}

				for _, r := range orgrepos {
					if repo.Filter.ExcludeForks {
						if r.ForkedFromID != 0 {
							continue
						}
					}
					urls, _, err := client.GetCloneUrl(r.ID)
					if err != nil {
						sub.Error().
							Msg("couldn't get clone urls")
						continue
					}

					defaultbranch, _, err := client.GetDefaultBranch(r.ID)
					if err != nil {
						sub.Error().
							Msgf("couldn't get default branch for %s", r.Name)
						defaultbranch = "main"
					}

					repos = append(repos, types.Repo{
						Name:          r.Name,
						URL:           urls.HTTP,
						SSHURL:        urls.SSH,
						Token:         repo.Token,
						Defaultbranch: defaultbranch,
						Origin:        repo,
						Owner:         org,
						Hoster:        types.GetHost(repo.URL),
						Description:   r.Description,
						Issues:        GetIssues(&r, client, repo, urls.HTTP),
					})
				}
			}
		}
	}

	return repos, ran
}

func GetOrCreate(destination types.GenRepo, repo types.Repo) (string, error) {
	client := &onedev.Client{}
	if destination.URL == "" {
		destination.URL = "https://code.onedev.io/"
	}
	sub = logger.CreateSubLogger("stage", "onedev", "url", destination.URL)

	if destination.Token != "" || destination.TokenFile != "" {
		client = onedev.NewClient(destination.URL, onedev.SetToken(destination.GetToken()))
	} else {
		if destination.Password != "" {
			client = onedev.NewClient(destination.URL, onedev.SetBasicAuth(destination.Username, destination.Password))
		} else {
			client = onedev.NewClient(destination.URL, "", "")
		}
	}

	dest := ""

	if destination.Organization == "" {
		user, _, err := client.GetMe()
		if err != nil {
			return "", err
		}
		dest = user.Name
	} else {
		dest = destination.Organization
	}

	query := onedev.ProjectQueryOptions{
		Query:  fmt.Sprintf("\"Name\" is \"%s\" and children of \"%s\"", repo.Name, dest),
		Offset: 0,
		Count:  100,
	}
	projects, _, err := client.GetProjects(&query)

	if err != nil {
		return "", err
	}

	for _, project := range projects {
		if project.Name == repo.Name {
			cloneUrls, _, err := client.GetCloneUrl(project.ID)
			if err != nil {
				return "", err
			}
			return cloneUrls.HTTP, nil
		}
	}

	query.Query = fmt.Sprintf("\"Name\" is \"%s\"", dest)

	parentid := 0
	parents, _, err := client.GetProjects(&query)
	if err != nil {
		return "", err
	}
	for _, parent := range parents {
		if parent.Name == dest {
			parentid = parent.ID
			break
		}
	}

	project, _, err := client.CreateProject(&onedev.CreateProjectOptions{Name: repo.Name, ParentID: parentid, CodeManagement: true})
	if err != nil {
		return "", err
	}

	cloneUrls, _, err := client.GetCloneUrl(project)
	if err != nil {
		return "", err
	}

	return cloneUrls.HTTP, nil
}

// GetIssues get issues
func GetIssues(repo *onedev.Project, client *onedev.Client, conf types.GenRepo, repourl string) map[string]interface{} {
	issues := map[string]interface{}{}
	if conf.Issues {
		name := strings.TrimPrefix(repourl, conf.URL)
		listOptions := &onedev.IssueQueryOptions{Count: 100, Offset: 0, Query: fmt.Sprintf("\"Project\" is \"%s\"", name)}
		for {
			i, _, err := client.GetIssues(listOptions)
			if err != nil {
				sub.Error().Err(err).Str("repo", repo.Name).Msg("can't fetch issues")
			} else {
				if len(i) > 0 {
					for _, issue := range i {
						onedevissue := Issue{Issue: issue}
						comments, _, err := client.GetIssueComments(onedevissue.ID)
						if err != nil {
							sub.Error().Err(err).Str("repo", repo.Name).Msg("can't fetch issues")
						} else {
							onedevissue.Comments = comments
						}

						issues[strconv.Itoa(int(issue.Number))] = onedevissue
					}
				} else {
					break
				}
				listOptions.Offset += listOptions.Count
			}
		}
	}
	return issues
}

type Issue struct {
	onedev.Issue
	Comments []onedev.Comment
}

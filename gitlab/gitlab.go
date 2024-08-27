package gitlab

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
	"github.com/xanzy/go-gitlab"
)

var (
	sub zerolog.Logger
)

// Backup TODO.
func Backup(r types.Repo, d types.GenRepo, dry bool) bool {
	var gitlabclient *gitlab.Client
	token := d.GetToken()
	var err error
	if d.URL == "" {
		d.URL = "https://gitlab.com"
		gitlabclient, err = gitlab.NewClient(token)
	} else {
		gitlabclient, err = gitlab.NewClient(token, gitlab.WithBaseURL(d.URL))
	}
	sub = logger.CreateSubLogger("stage", "gitlab", "url", d.URL)

	if err != nil {
		sub.Error().
			Msg(err.Error())
		return false
	}

	sub.Info().
		Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

	True := true

	opt := gitlab.ListProjectsOptions{
		Search: &r.Name,
		Owned:  &True,
	}

	projects, _, err := gitlabclient.Projects.ListProjects(&opt)
	if err != nil {
		sub.Error().Msg(err.Error())
		return false
	}

	found := false
	for _, p := range projects {
		if p.Name == r.Name {
			found = true
		}
	}

	if dry || found {
		return true
	}

	if r.Token != "" {
		splittedurl := strings.Split(r.URL, "//")

		r.URL = fmt.Sprintf("%s//%s:%s@%s",
			splittedurl[0], r.Owner, r.Token, splittedurl[1])
	}

	var visibility gitlab.VisibilityValue

	if r.Private {
		visibility = gitlab.PrivateVisibility
	} else {
		visibility = gitlab.PublicVisibility
	}

	opts := &gitlab.CreateProjectOptions{
		Mirror:      &True,
		ImportURL:   &r.URL,
		Name:        &r.Name,
		Description: &r.Description,
		Visibility:  gitlab.Ptr(visibility),
	}

	_, _, err = gitlabclient.Projects.CreateProject(opts)
	if err != nil {
		sub.Error().
			Msg(err.Error())
		return false
	}

	return true
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gitlab {
		if repo.URL == "" {
			repo.URL = "https://gitlab.com"
		}
		err := repo.Filter.ParseDuration()
		sub = logger.CreateSubLogger("stage", "gitlab", "url", repo.URL)
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}
		ran = true

		token := repo.GetToken()
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(repo.URL))
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		if repo.User == "" {
			user, _, err := client.Users.CurrentUser()
			if err != nil {
				sub.Error().
					Msg(err.Error())
				continue
			}
			repo.User = user.Username
		}

		sub.Info().
			Msgf("grabbing repositories from %s", repo.User)
		gitlabrepos := []*gitlab.Project{}
		gitlabgrouprepos := map[string][]*gitlab.Project{}

		opt := &gitlab.ListProjectsOptions{}
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &repo.User})
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		opt.PerPage = 50
		for _, user := range users {
			if user.Username == repo.User {
				i := 1
				for {
					opt.Page = i
					projects, _, err := client.Projects.ListUserProjects(user.ID, opt)
					if err != nil {
						sub.Error().
							Msg(err.Error())
					}
					if len(projects) == 0 {
						break
					}
					gitlabrepos = append(gitlabrepos, projects...)
					i++
				}
			}
		}

		if repo.Starred {
			for _, user := range users {
				if user.Username == repo.User {
					i := 1
					for {
						opt.Page = i
						projects, _, err := client.Projects.ListUserStarredProjects(user.ID, opt)
						if err != nil {
							sub.Error().
								Msg(err.Error())
						}
						if len(projects) == 0 {
							break
						}
						gitlabrepos = append(gitlabrepos, projects...)
						i++
					}
				}
			}
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)
		languages := types.GetMap(repo.Filter.Languages)

		for _, org := range repo.IncludeOrgs {
			group, _, err := client.Groups.GetGroup(org, &gitlab.GetGroupOptions{})
			if err != nil {
				sub.Error().
					Msg(err.Error())
				continue
			}
			subgroups, _, err := client.Groups.ListSubGroups(group.ID, &gitlab.ListSubGroupsOptions{})
			for _, sub := range subgroups {
				includeorgs[sub.FullPath] = true
			}
		}

		for _, org := range repo.ExcludeOrgs {
			group, _, err := client.Groups.GetGroup(org, &gitlab.GetGroupOptions{})
			if err != nil {
				sub.Error().
					Msg(err.Error())
				continue
			}
			subgroups, _, err := client.Groups.ListSubGroups(group.ID, &gitlab.ListSubGroupsOptions{})
			for _, sub := range subgroups {
				excludeorgs[sub.FullPath] = true
			}
		}

		for _, r := range gitlabrepos {
			if repo.Filter.ExcludeForks {
				if r.ForkedFromProject != nil {
					continue
				}
			}
			if repo.Filter.ExcludeArchived {
				if r.Archived {
					continue
				}
			}

			if len(repo.Filter.Languages) > 0 {
				sub.Debug().Msg(r.WebURL)
				langs, _, err := client.Projects.GetProjectLanguages(r.ID)
				if err != nil {
					sub.Error().
						Msg(err.Error())
					continue
				} else {
					language := ""
					percentage := float32(0)

					for lang, percent := range *langs {
						if percent > percentage {
							language = lang
						}
					}
					if !languages[strings.ToLower(language)] {
						continue
					}
				}
			}

			if r.StarCount < repo.Filter.Stars {
				continue
			}
			if time.Since(*r.LastActivityAt) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
				continue
			}
			if include[r.Name] {
				if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
					repos = append(repos, types.Repo{
						Name:          r.Path,
						URL:           r.HTTPURLToRepo,
						SSHURL:        r.SSHURLToRepo,
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Namespace.FullPath,
						Hoster:        types.GetHost(repo.URL),
						Description:   r.Description,
						Private:       r.Visibility == gitlab.PrivateVisibility,
						Issues:        GetIssues(r, client, repo),
					})
				}

				if r.WikiEnabled && repo.Wiki {
					if activeWiki(r, client, repo) {
						httpURLToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
						sshURLToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
						repos = append(repos, types.Repo{
							Name:          r.Path + ".wiki",
							URL:           httpURLToRepo,
							SSHURL:        sshURLToRepo,
							Token:         token,
							Defaultbranch: r.DefaultBranch,
							Origin:        repo,
							Owner:         r.Namespace.FullPath,
							Hoster:        types.GetHost(repo.URL),
							Description:   r.Description,
							Private:       r.Visibility == gitlab.PrivateVisibility,
						})
					}
				}

				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(include) == 0 {
				if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
					repos = append(repos, types.Repo{
						Name:          r.Path,
						URL:           r.HTTPURLToRepo,
						SSHURL:        r.SSHURLToRepo,
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Namespace.FullPath,
						Hoster:        types.GetHost(repo.URL),
						Description:   r.Description,
						Private:       r.Visibility == gitlab.PrivateVisibility,
						Issues:        GetIssues(r, client, repo),
					})
				}

				if r.WikiEnabled && repo.Wiki {
					if activeWiki(r, client, repo) {
						httpURLToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
						sshURLToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
						repos = append(repos, types.Repo{
							Name:          r.Path + ".wiki",
							URL:           httpURLToRepo,
							SSHURL:        sshURLToRepo,
							Token:         token,
							Defaultbranch: r.DefaultBranch,
							Origin:        repo,
							Owner:         r.Namespace.FullPath,
							Hoster:        types.GetHost(repo.URL),
							Description:   r.Description,
							Private:       r.Visibility == gitlab.PrivateVisibility,
						})
					}
				}
			}
		}

		if token != "" {
			groups := []*gitlab.Group{}
			i := 1
			for {
				g, _, err := client.Groups.ListGroups(&gitlab.ListGroupsOptions{
					ListOptions: gitlab.ListOptions{
						Page:    i,
						PerPage: 50,
					},
				})
				if err != nil {
					sub.Error().Msg(err.Error())
				}

				if len(g) == 0 {
					break
				}

				groups = append(groups, g...)
				i++
			}

			gopt := &gitlab.ListGroupProjectsOptions{}
			for _, group := range groups {
				i = 1
				gopt.PerPage = 50
				gopt.Page = i
				for {
					projects, _, err := client.Groups.ListGroupProjects(group.ID, gopt)
					if err != nil {
						sub.Error().
							Msg(err.Error())
					}
					if len(projects) == 0 {
						break
					}
					splfullpath := strings.Split(group.FullPath, "/")
					fullpath := path.Join(splfullpath...)
					if _, ok := gitlabgrouprepos[fullpath]; !ok {
						gitlabgrouprepos[fullpath] = []*gitlab.Project{}
					}

					gitlabgrouprepos[fullpath] = append(gitlabgrouprepos[fullpath], projects...)
					i++
					gopt.Page = i
				}
			}
			for k, gr := range gitlabgrouprepos {
				for _, r := range gr {
					if repo.Filter.ExcludeForks {
						if r.ForkedFromProject != nil {
							continue
						}
					}
					if repo.Filter.ExcludeArchived {
						if r.Archived {
							continue
						}
					}

					if len(repo.Filter.Languages) > 0 {
						langs, _, err := client.Projects.GetProjectLanguages(r.ID)
						if err != nil {
							sub.Error().
								Msg(err.Error())
							continue
						} else {
							language := ""
							percentage := float32(0)

							for lang, percent := range *langs {
								if percent > percentage {
									language = lang
								}
							}
							if !languages[strings.ToLower(language)] {
								continue
							}
						}
					}

					if r.StarCount < repo.Filter.Stars {
						continue
					}
					if time.Since(*r.LastActivityAt) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
						continue
					}

					if include[r.Name] {
						if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
							repos = append(repos, types.Repo{
								Name:          r.Path,
								URL:           r.HTTPURLToRepo,
								SSHURL:        r.SSHURLToRepo,
								Token:         token,
								Defaultbranch: r.DefaultBranch,
								Origin:        repo,
								Owner:         k,
								Hoster:        types.GetHost(repo.URL),
								Description:   r.Description,
								Private:       r.Visibility == gitlab.PrivateVisibility,
								Issues:        GetIssues(r, client, repo),
							})
						}

						if r.WikiEnabled && repo.Wiki {
							if activeWiki(r, client, repo) {
								httpURLToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
								sshURLToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
								repos = append(repos, types.Repo{
									Name:          r.Path + ".wiki",
									URL:           httpURLToRepo,
									SSHURL:        sshURLToRepo,
									Token:         token,
									Defaultbranch: r.DefaultBranch,
									Origin:        repo,
									Owner:         k,
									Hoster:        types.GetHost(repo.URL),
									Description:   r.Description,
									Private:       r.Visibility == gitlab.PrivateVisibility,
								})
							}
						}

						continue
					}
					if exclude[r.Name] {
						continue
					}
					if excludeorgs[r.Namespace.FullPath] {
						continue
					}

					if len(include) == 0 {
						if len(includeorgs) == 0 || includeorgs[r.Namespace.FullPath] {
							if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
								repos = append(repos, types.Repo{
									Name:          r.Path,
									URL:           r.HTTPURLToRepo,
									SSHURL:        r.SSHURLToRepo,
									Token:         token,
									Defaultbranch: r.DefaultBranch,
									Origin:        repo,
									Owner:         k,
									Hoster:        types.GetHost(repo.URL),
									Description:   r.Description,
									Private:       r.Visibility == gitlab.PrivateVisibility,
									Issues:        GetIssues(r, client, repo),
								})
							}

							if r.WikiEnabled && repo.Wiki {
								if activeWiki(r, client, repo) {
									httpURLToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
									sshURLToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
									repos = append(repos, types.Repo{
										Name:          r.Path + ".wiki",
										URL:           httpURLToRepo,
										SSHURL:        sshURLToRepo,
										Token:         token,
										Defaultbranch: r.DefaultBranch,
										Origin:        repo,
										Owner:         k,
										Hoster:        types.GetHost(repo.URL),
										Description:   r.Description,
										Private:       r.Visibility == gitlab.PrivateVisibility,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return repos, ran
}

func activeWiki(r *gitlab.Project, client *gitlab.Client, repo types.GenRepo) bool {
	wikilistoptions := &gitlab.ListWikisOptions{
		WithContent: gitlab.Ptr(true),
	}

	wikis, _, err := client.Wikis.ListWikis(r.ID, wikilistoptions)
	if err != nil {
		sub.Warn().
			Msg(err.Error())
	}

	return len(wikis) > 0
}

// GetIssues get issues
func GetIssues(repo *gitlab.Project, client *gitlab.Client, conf types.GenRepo) map[string]interface{} {
	issues := map[string]interface{}{}
	if conf.Issues {
		listOptions := &gitlab.ListProjectIssuesOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
		errorcount := 0
		for {
			i, response, err := client.Issues.ListProjectIssues(repo.ID, listOptions)
			if err != nil {
				if response.StatusCode == http.StatusForbidden {
					sub.Error().Err(err).Str("repo", repo.Name).Msg("can't fetch issues")
					return issues
				}
				if errorcount < 5 {
					sub.Error().Err(err).Str("repo", repo.Name).Msg("can't fetch issues")
					time.Sleep(5 * time.Second)
					errorcount++
				} else {
					return issues
				}
			} else {
				if len(i) > 0 {
					for _, issue := range i {
						issues[strconv.Itoa(issue.IID)] = issue
					}
				} else {
					break
				}
				listOptions.Page++
			}
		}
	}
	return issues
}

// GetOrCreate Get or create a repository
func GetOrCreate(destination types.GenRepo, repo types.Repo) (string, error) {
	visibility := gitlab.PublicVisibility

	if repo.Private {
		visibility = gitlab.PrivateVisibility
	}

	sub = logger.CreateSubLogger("stage", "gitlab", "url", destination.URL)

	token := destination.GetToken()
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(destination.URL))
	if err != nil {
		return "", err
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return "", err
	}
	me := user

	if destination.User == "" {
		destination.User = me.Username
	}

	repos, _, err := client.Projects.ListProjects(&gitlab.ListProjectsOptions{Search: &repo.Name})

	if err != nil {
		return "", err
	}

	for _, repository := range repos {
		if repository.Owner == nil {
			continue
		}
		if repository.Name == repo.Name && repository.Owner.Username == me.Username {
			return repository.HTTPURLToRepo, nil
		}
	}

	opts := gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(repo.Name),
		Visibility:  gitlab.Ptr(visibility),
		Description: gitlab.Ptr(repo.Description),
	}

	r, _, err := client.Projects.CreateProject(&opts)
	if err != nil {
		return "", err
	}

	return r.HTTPURLToRepo, nil
}

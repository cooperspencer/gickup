package gitlab

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
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

	if err != nil {
		log.Error().
			Str("stage", "gitlab").
			Str("url", d.URL).
			Msg(err.Error())
		return false
	}

	log.Info().
		Str("stage", "gitlab").
		Str("url", d.URL).
		Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

	True := true

	opt := gitlab.ListProjectsOptions{
		Search: &r.Name,
		Owned:  &True,
	}

	projects, _, err := gitlabclient.Projects.ListProjects(&opt)
	if err != nil {
		log.Error().Str("stage", "gitlab").Str("url", d.URL).Msg(err.Error())
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
			splittedurl[0], r.Origin.User, r.Origin.Password, splittedurl[1])
	}

	opts := &gitlab.CreateProjectOptions{
		Mirror:    &True,
		ImportURL: &r.URL,
		Name:      &r.Name,
	}

	_, _, err = gitlabclient.Projects.CreateProject(opts)
	if err != nil {
		log.Error().
			Str("stage", "gitlab").
			Str("url", d.URL).
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
		err := repo.Filter.ParseDuration()
		if err != nil {
			log.Error().
				Str("stage", "bitbucket").
				Str("url", repo.URL).
				Msg(err.Error())
		}
		ran = true
		if repo.URL == "" {
			repo.URL = "https://gitlab.com"
		}

		log.Info().
			Str("stage", "gitlab").
			Str("url", repo.URL).
			Msgf("grabbing repositories from %s", repo.User)
		gitlabrepos := []*gitlab.Project{}
		gitlabgrouprepos := map[string][]*gitlab.Project{}
		token := repo.GetToken()
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(repo.URL))
		if err != nil {
			log.Fatal().
				Str("stage", "gitlab").
				Str("url", repo.URL).
				Msg(err.Error())
		}

		opt := &gitlab.ListProjectsOptions{}
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &repo.User})
		if err != nil {
			log.Fatal().
				Str("stage", "gitlab").
				Str("url", repo.URL).
				Msg(err.Error())
		}

		opt.PerPage = 50
		for _, user := range users {
			if user.Username == repo.User {
				i := 1
				for {
					opt.Page = i
					projects, _, err := client.Projects.ListUserProjects(user.ID, opt)
					if err != nil {
						log.Fatal().
							Str("stage", "gitlab").
							Str("url", repo.URL).
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
							log.Fatal().
								Str("stage", "gitlab").
								Str("url", repo.URL).
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

		for _, r := range gitlabrepos {
			if repo.Filter.ExcludeArchived {
				if r.Archived {
					continue
				}
			}

			if len(repo.Filter.Languages) > 0 {
				langs, _, err := client.Projects.GetProjectLanguages(r.ID)
				if err != nil {
					log.Error().
						Str("stage", "gitlab").
						Str("url", repo.URL).
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
					log.Fatal().
						Str("stage", "gitlab").
						Str("url", repo.URL).Msg(err.Error())
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
						log.Fatal().
							Str("stage", "gitlab").
							Str("url", repo.URL).
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
					if repo.Filter.ExcludeArchived {
						if r.Archived {
							continue
						}
					}

					if len(repo.Filter.Languages) > 0 {
						langs, _, err := client.Projects.GetProjectLanguages(r.ID)
						if err != nil {
							log.Error().
								Str("stage", "gitlab").
								Str("url", repo.URL).
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
								})
							}

							if r.WikiEnabled && repo.Wiki {
								if activeWiki(r, client, repo) {
									httpURLToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
									sshURLToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
									repos = append(repos, types.Repo{
										Name:   r.Path + ".wiki",
										URL:    httpURLToRepo,
										SSHURL: sshURLToRepo,
										Token:  token, Defaultbranch: r.DefaultBranch,
										Origin: repo,
										Owner:  k,
										Hoster: types.GetHost(repo.URL),
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
		WithContent: gitlab.Bool(true),
	}

	wikis, _, err := client.Wikis.ListWikis(r.ID, wikilistoptions)
	if err != nil {
		log.Warn().
			Str("stage", "gitlab").
			Str("url", repo.URL).
			Msg(err.Error())
	}

	return len(wikis) > 0
}

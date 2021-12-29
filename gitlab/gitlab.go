package gitlab

import (
	"fmt"
	"gickup/types"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func Backup(r types.Repo, d types.GenRepo, dry bool) {
	gitlabclient := &gitlab.Client{}
	token := d.GetToken()
	var err error
	if d.Url == "" {
		d.Url = "https://gitlab.com"
		gitlabclient, err = gitlab.NewClient(token)
	} else {
		gitlabclient, err = gitlab.NewClient(token, gitlab.WithBaseURL(d.Url))
	}
	log.Info().Str("stage", "gitlab").Str("url", d.Url).Msgf("mirroring %s to %s", types.Blue(r.Name), d.Url)
	if err != nil {
		log.Fatal().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
	}

	True := true
	opt := gitlab.ListProjectsOptions{Search: &r.Name, Owned: &True}
	projects, _, err := gitlabclient.Projects.ListProjects(&opt)
	if err != nil {
		log.Fatal().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
	}

	found := false
	for _, p := range projects {
		if p.Name == r.Name {
			found = true
		}
	}

	if !dry {
		if !found {
			if r.Token != "" {
				splittedurl := strings.Split(r.Url, "//")
				r.Url = fmt.Sprintf("%s//%s@%s", splittedurl[0], r.Token, splittedurl[1])
				if r.Token == "" {
					r.Url = fmt.Sprintf("%s//%s:%s@%s", splittedurl[0], r.Origin.User, r.Origin.Password, splittedurl[1])
				}
			}
			opts := &gitlab.CreateProjectOptions{Mirror: &True, ImportURL: &r.Url, Name: &r.Name}
			_, _, err := gitlabclient.Projects.CreateProject(opts)
			if err != nil {
				log.Fatal().Str("stage", "gitlab").Str("url", d.Url).Msg(err.Error())
			}
		}
	}
}

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gitlab {
		if repo.Url == "" {
			repo.Url = "https://gitlab.com"
		}
		log.Info().Str("stage", "gitlab").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		gitlabrepos := []*gitlab.Project{}
		gitlabgrouprepos := map[string][]*gitlab.Project{}
		token := repo.GetToken()
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(repo.Url))
		if err != nil {
			log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
		}
		opt := &gitlab.ListProjectsOptions{}
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &repo.User})
		if err != nil {
			log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
		}

		opt.PerPage = 50
		i := 0
		for _, user := range users {
			if user.Username == repo.User {
				for {
					projects, _, err := client.Projects.ListUserProjects(user.ID, opt)
					if err != nil {
						log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
					}
					if len(projects) == 0 {
						break
					}
					gitlabrepos = append(gitlabrepos, projects...)
					i++
					opt.Page = i
				}
			}
		}
		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)

		for _, r := range gitlabrepos {
			if include[r.Name] {
				if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
					repos = append(repos, types.Repo{Name: r.Path, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.Username, Hoster: types.GetHost(repo.Url)})
				}

				if r.WikiEnabled && repo.Wiki {
					httpUrlToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
					sshUrlToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
					repos = append(repos, types.Repo{Name: r.Path + ".wiki", Url: httpUrlToRepo, SshUrl: sshUrlToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.Username, Hoster: types.GetHost(repo.Url)})
				}

				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(include) == 0 {
				if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
					repos = append(repos, types.Repo{Name: r.Path, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.Username, Hoster: types.GetHost(repo.Url)})
				}

				if r.WikiEnabled && repo.Wiki {
					httpUrlToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
					sshUrlToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
					repos = append(repos, types.Repo{Name: r.Path + ".wiki", Url: httpUrlToRepo, SshUrl: sshUrlToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.Username, Hoster: types.GetHost(repo.Url)})
				}
			}
		}
		if token != "" {
			groups := []*gitlab.Group{}
			i = 1
			for {
				g, _, err := client.Groups.ListGroups(&gitlab.ListGroupsOptions{ListOptions: gitlab.ListOptions{Page: i, PerPage: 50}})
				if err != nil {
					log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
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
						log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
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
					if include[r.Name] {
						if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
							repos = append(repos, types.Repo{Name: r.Path, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: k, Hoster: types.GetHost(repo.Url)})
						}

						if r.WikiEnabled && repo.Wiki {
							httpUrlToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
							sshUrlToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
							repos = append(repos, types.Repo{Name: r.Path + ".wiki", Url: httpUrlToRepo, SshUrl: sshUrlToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: k, Hoster: types.GetHost(repo.Url)})
						}
						continue
					}
					if exclude[r.Name] {
						continue
					}
					if len(include) == 0 {
						if r.RepositoryAccessLevel != gitlab.DisabledAccessControl {
							repos = append(repos, types.Repo{Name: r.Path, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: k, Hoster: types.GetHost(repo.Url)})
						}

						if r.WikiEnabled && repo.Wiki {
							httpUrlToRepo := types.DotGitRx.ReplaceAllString(r.HTTPURLToRepo, ".wiki.git")
							sshUrlToRepo := types.DotGitRx.ReplaceAllString(r.SSHURLToRepo, ".wiki.git")
							repos = append(repos, types.Repo{Name: r.Path + ".wiki", Url: httpUrlToRepo, SshUrl: sshUrlToRepo, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: k, Hoster: types.GetHost(repo.Url)})
						}
					}
				}
			}
		}
	}
	return repos
}

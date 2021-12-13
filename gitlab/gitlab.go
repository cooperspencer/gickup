package gitlab

import (
	"fmt"
	"gickup/types"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func Backup(r types.Repo, d types.GenRepo, dry bool) {
	gitlabclient := &gitlab.Client{}
	var err error
	if d.Url == "" {
		d.Url = "https://gitlab.com"
		gitlabclient, err = gitlab.NewClient(d.Token)
	} else {
		gitlabclient, err = gitlab.NewClient(d.Token, gitlab.WithBaseURL(d.Url))
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
		gitlabgrouprepos := []*gitlab.Project{}
		client, err := gitlab.NewClient(repo.Token, gitlab.WithBaseURL(repo.Url))
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

		exclude := types.GetExcludedMap(repo.Exclude)

		for _, r := range gitlabrepos {
			if exclude[r.Name] {
				continue
			}
			repos = append(repos, types.Repo{Name: r.Name, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo})
		}
		groups, _, err := client.Groups.ListGroups(&gitlab.ListGroupsOptions{})
		if err != nil {
			log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
		}

		visibilities := []gitlab.VisibilityValue{gitlab.PrivateVisibility, gitlab.PublicVisibility, gitlab.InternalVisibility}

		for _, visibility := range visibilities {
			gopt := &gitlab.ListGroupProjectsOptions{Visibility: gitlab.Visibility(visibility)}
			gopt.PerPage = 50
			i = 0
			for _, group := range groups {
				for {
					projects, _, err := client.Groups.ListGroupProjects(group.ID, gopt)
					if err != nil {
						log.Fatal().Str("stage", "gitlab").Str("url", repo.Url).Msg(err.Error())
					}
					if len(projects) == 0 {
						break
					}
					gitlabgrouprepos = append(gitlabgrouprepos, projects...)
					i++
					gopt.Page = i
				}
			}
			for _, r := range gitlabgrouprepos {
				if exclude[r.Name] {
					continue
				}
				repos = append(repos, types.Repo{Name: r.Name, Url: r.HTTPURLToRepo, SshUrl: r.SSHURLToRepo, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo})
			}
		}
	}
	return repos
}

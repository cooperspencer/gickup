package github

import (
	"context"
	"gickup/types"

	"github.com/google/go-github/v41/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Github {
		log.Info().Str("stage", "github").Str("url", "https://github.com").Msgf("grabbing the repositories from %s", repo.User)
		client := &github.Client{}
		opt := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 50}}
		i := 1
		githubrepos := []*github.Repository{}
		token := repo.GetToken()
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
				log.Fatal().Str("stage", "github").Str("url", "https://github.com").Msg(err.Error())
			}
			if repo.User == user.GetLogin() {
				repo.User = ""
			}
		}
		for {
			opt.Page = i
			repos, _, err := client.Repositories.List(context.TODO(), repo.User, opt)
			if err != nil {
				log.Fatal().Str("stage", "github").Str("url", "https://github.com").Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			githubrepos = append(githubrepos, repos...)
			i++
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range githubrepos {
			if include[*r.Name] {
				repos = append(repos, types.Repo{Name: r.GetName(), Url: r.GetCloneURL(), SshUrl: r.GetSSHURL(), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
				if *r.HasWiki && repo.Wiki && *r.HasPages {
					repos = append(repos, types.Repo{Name: *r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.GetCloneURL(), ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.GetSSHURL(), ".wiki.git"), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
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
						repos = append(repos, types.Repo{Name: r.GetName(), Url: r.GetCloneURL(), SshUrl: r.GetSSHURL(), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
						if *r.HasWiki && repo.Wiki && *r.HasPages {
							repos = append(repos, types.Repo{Name: *r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.GetCloneURL(), ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.GetSSHURL(), ".wiki.git"), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
						}
					}
				} else {
					repos = append(repos, types.Repo{Name: r.GetName(), Url: r.GetCloneURL(), SshUrl: r.GetSSHURL(), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
					if *r.HasWiki && repo.Wiki && *r.HasPages {
						repos = append(repos, types.Repo{Name: *r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.GetCloneURL(), ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.GetSSHURL(), ".wiki.git"), Token: token, Defaultbranch: r.GetDefaultBranch(), Origin: repo, Owner: r.GetOwner().GetLogin(), Hoster: "github.com"})
					}
				}
			}
		}
	}
	return repos
}

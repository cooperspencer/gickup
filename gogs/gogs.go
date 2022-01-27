package gogs

import (
	"gickup/types"

	"github.com/gogs/go-gogs-client"
	"github.com/rs/zerolog/log"
)

func Backup(r types.Repo, d types.GenRepo, dry bool) {
	log.Info().Str("stage", "gogs").Str("url", d.Url).Msgf("mirroring %s to %s", types.Blue(r.Name), d.Url)
	gogsclient := gogs.NewClient(d.Url, d.GetToken())

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		log.Fatal().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
	}
	if !dry {
		repo, err := gogsclient.GetRepo(user.UserName, r.Name)
		if err != nil {
			opts := gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Token}
			if r.Token == "" {
				opts = gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Origin.User, AuthPassword: r.Origin.Password}
			}
			_, err := gogsclient.MigrateRepo(opts)
			if err != nil {
				log.Fatal().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
			}
		} else {
			if repo.Mirror {
				log.Info().Str("stage", "gogs").Str("url", d.Url).Msgf("mirror of %s already exists, syncing instead", types.Blue(r.Name))
				err := gogsclient.MirrorSync(user.UserName, repo.Name)
				if err != nil {
					log.Fatal().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
				}
				log.Info().Str("stage", "gogs").Str("url", d.Url).Msgf("successfully synced %s.", types.Blue(r.Name))
			}
		}
	}
}

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gogs {
		log.Info().Str("stage", "gogs").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		token := repo.GetToken()
		client := gogs.NewClient(repo.Url, token)
		gogsrepos, err := client.ListUserRepos(repo.User)
		if err != nil {
			log.Fatal().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range gogsrepos {
			if include[r.Name] {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(include) == 0 {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
			}
		}
		orgs, err := client.ListUserOrgs(repo.User)
		if err != nil {
			log.Fatal().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
		}

		orgrepos := []*gogs.Repository{}
		for _, org := range orgs {
			if excludeorgs[org.UserName] {
				continue
			}
			for {
				if len(includeorgs) > 0 {
					if includeorgs[org.UserName] {
						o, err := client.ListOrgRepos(org.UserName)
						if err != nil {
							log.Fatal().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
						}
						if len(o) == 0 {
							break
						}
						orgrepos = append(orgrepos, o...)
					}
				} else {
					o, err := client.ListOrgRepos(org.UserName)
					if err != nil {
						log.Fatal().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
					}
					if len(o) == 0 {
						break
					}
					orgrepos = append(orgrepos, o...)
				}
			}
		}
		for _, r := range orgrepos {
			if include[r.Name] {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(repo.Include) == 0 {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
			}
		}
	}
	return repos
}

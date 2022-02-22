package gitea

import (
	"code.gitea.io/sdk/gitea"
	"gickup/types"
	"github.com/rs/zerolog/log"
)

func Backup(r types.Repo, d types.GenRepo, dry bool) {
	if d.Url == "" {
		d.Url = "https://gitea.com/"
	}
	log.Info().Str("stage", "gitea").Str("url", d.Url).Msgf("mirroring %s to %s", types.Blue(r.Name), d.Url)
	giteaclient, err := gitea.NewClient(d.Url)
	if err != nil {
		log.Fatal().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
	}
	giteaclient.SetBasicAuth(d.GetToken(), "")
	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		log.Fatal().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
	}
	if !dry {
		repo, _, err := giteaclient.GetRepo(user.UserName, r.Name)
		if err != nil {
			opts := gitea.MigrateRepoOption{RepoName: r.Name, RepoOwner: user.UserName, Mirror: true, CloneAddr: r.Url, AuthToken: r.Token}
			if r.Token == "" {
				opts = gitea.MigrateRepoOption{RepoName: r.Name, RepoOwner: user.UserName, Mirror: true, CloneAddr: r.Url, AuthUsername: r.Origin.User, AuthPassword: r.Origin.Password}
			}
			_, _, err := giteaclient.MigrateRepo(opts)
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
			}
			log.Info().Str("stage", "gitea").Str("url", d.Url).Msgf("mirrored %s to %s", types.Blue(r.Name), d.Url)
		} else {
			if repo.Mirror {
				log.Info().Str("stage", "gitea").Str("url", d.Url).Msgf("mirror of %s already exists, syncing instead", types.Blue(r.Name))
				_, err := giteaclient.MirrorSync(user.UserName, repo.Name)
				if err != nil {
					log.Fatal().Str("stage", "gitea").Str("url", d.Url).Msg(err.Error())
				}
				log.Info().Str("stage", "gitea").Str("url", d.Url).Msgf("successfully synced %s.", types.Blue(r.Name))
			}
		}
	}
}

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gitea {
		if repo.Url == "" {
			repo.Url = "https://gitea.com"
		}
		log.Info().Str("stage", "gitea").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		opt := gitea.ListReposOptions{}
		opt.PageSize = 50
		opt.Page = 1
		gitearepos := []*gitea.Repository{}
		client := &gitea.Client{}
		var err error
		token := repo.GetToken()
		if token != "" {
			client, err = gitea.NewClient(repo.Url, gitea.SetToken(token))
		} else {
			client, err = gitea.NewClient(repo.Url)
		}
		for {
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			repos, _, err := client.ListUserRepos(repo.User, opt)
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			gitearepos = append(gitearepos, repos...)
			opt.Page++
		}

		if repo.Starred {
			starredrepos, _, err := client.GetMyStarredRepos()
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			gitearepos = append(gitearepos, starredrepos...)
		}

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range gitearepos {
			if include[r.Name] {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if r.HasWiki && repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(repo.Include) == 0 {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if r.HasWiki && repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
			}
		}
		orgopt := gitea.ListOptions{Page: 1, PageSize: 50}
		orgs := []*gitea.Organization{}
		for {
			o, _, err := client.ListUserOrgs(repo.User, gitea.ListOrgsOptions{ListOptions: orgopt})
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if len(o) == 0 {
				break
			}
			orgs = append(orgs, o...)
			orgopt.Page++
		}

		orgrepos := []*gitea.Repository{}
		for _, org := range orgs {
			orgopt.Page = 1
			if excludeorgs[org.UserName] {
				continue
			}
			for {
				if len(includeorgs) > 0 {
					if includeorgs[org.UserName] {
						o := getOrgRepos(client, org, orgopt, repo)
						if len(o) == 0 {
							break
						}
						orgrepos = append(orgrepos, o...)
					}
				} else {
					o := getOrgRepos(client, org, orgopt, repo)
					if len(o) == 0 {
						break
					}
					orgrepos = append(orgrepos, o...)
				}
				orgopt.Page++
			}
		}
		for _, r := range orgrepos {
			if include[r.Name] {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if r.HasWiki && repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(repo.Include) == 0 {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				if r.HasWiki && repo.Wiki {
					repos = append(repos, types.Repo{Name: r.Name + ".wiki", Url: types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"), SshUrl: types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"), Token: token, Defaultbranch: r.DefaultBranch, Origin: repo, Owner: r.Owner.UserName, Hoster: types.GetHost(repo.Url)})
				}
			}
		}
	}
	return repos
}

func getOrgRepos(client *gitea.Client, org *gitea.Organization, orgopt gitea.ListOptions, repo types.GenRepo) []*gitea.Repository {
	o, _, err := client.ListOrgRepos(org.UserName, gitea.ListOrgReposOptions{orgopt})
	if err != nil {
		log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
	}
	return o
}

package gitea

import (
	"code.gitea.io/sdk/gitea"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

// Backup TODO.
func Backup(r types.Repo, d types.GenRepo, dry bool) {
	if d.URL == "" {
		d.URL = "https://gitea.com/"
	}

	log.Info().
		Str("stage", "gitea").
		Str("url", d.URL).
		Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

	giteaclient, err := gitea.NewClient(d.URL, gitea.SetToken(d.GetToken()))
	if err != nil {
		log.Fatal().Str("stage", "gitea").Str("url", d.URL).Msg(err.Error())
	}

	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		log.Fatal().
			Str("stage", "gitea").
			Str("url", d.URL).
			Msg(err.Error())
	}

	if d.User != "" {
		user, _, err = giteaclient.GetUserInfo(d.User)
		if err != nil {
			if d.CreateOrg {
				_, _, err = giteaclient.CreateOrg(gitea.CreateOrgOption{
					Name:       d.User,
					Visibility: gitea.VisibleTypePrivate,
				})
				if err != nil {
					log.Fatal().
						Str("stage", "gitea").
						Str("url", d.URL).
						Msg(err.Error())
				}
			} else {
				log.Fatal().
					Str("stage", "gitea").
					Str("url", d.URL).
					Msg(err.Error())
			}
		}

	}

	if dry {
		return
	}

	repo, _, err := giteaclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		opts := gitea.MigrateRepoOption{
			RepoName:  r.Name,
			RepoOwner: user.UserName,
			Mirror:    true,
			CloneAddr: r.URL,
			AuthToken: r.Token,
		}

		if r.Token == "" {
			opts = gitea.MigrateRepoOption{
				RepoName:     r.Name,
				RepoOwner:    user.UserName,
				Mirror:       true,
				CloneAddr:    r.URL,
				AuthUsername: r.Origin.User,
				AuthPassword: r.Origin.Password,
			}
		}

		_, _, err := giteaclient.MigrateRepo(opts)
		if err != nil {
			log.Fatal().
				Str("stage", "gitea").
				Str("url", d.URL).
				Msg(err.Error())
		}

		log.Info().
			Str("stage", "gitea").
			Str("url", d.URL).
			Msgf("mirrored %s to %s", types.Blue(r.Name), d.URL)

		return
	}
	if repo.Mirror {
		log.Info().
			Str("stage", "gitea").
			Str("url", d.URL).
			Msgf("mirror of %s already exists, syncing instead", types.Blue(r.Name))

		_, err := giteaclient.MirrorSync(user.UserName, repo.Name)
		if err != nil {
			log.Fatal().
				Str("stage", "gitea").
				Str("url", d.URL).
				Msg(err.Error())
		}

		log.Info().
			Str("stage", "gitea").
			Str("url", d.URL).
			Msgf("successfully synced %s.", types.Blue(r.Name))
	}
}

// Get TODO.
func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gitea {
		if repo.URL == "" {
			repo.URL = "https://gitea.com"
		}
		log.Info().
			Str("stage", "gitea").
			Str("url", repo.URL).
			Msgf("grabbing repositories from %s", repo.User)
		opt := gitea.ListReposOptions{}
		opt.PageSize = 50
		opt.Page = 1
		gitearepos := []*gitea.Repository{}

		var client *gitea.Client
		var err error
		token := repo.GetToken()
		if token != "" {
			client, err = gitea.NewClient(repo.URL, gitea.SetToken(token))
		} else {
			client, err = gitea.NewClient(repo.URL)
		}

		for {
			if err != nil {
				log.Fatal().
					Str("stage", "gitea").
					Str("url", repo.URL).
					Msg(err.Error())
			}
			repos, _, err := client.ListUserRepos(repo.User, opt)
			if err != nil {
				log.Fatal().
					Str("stage", "gitea").
					Str("url", repo.URL).
					Msg(err.Error())
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
				log.Fatal().
					Str("stage", "gitea").
					Str("url", repo.URL).
					Msg(err.Error())
			}
			gitearepos = append(gitearepos, starredrepos...)
		}

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range gitearepos {
			if include[r.Name] {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.CloneURL,
					SSHURL:        r.SSHURL,
					Token:         token,
					Defaultbranch: r.DefaultBranch,
					Origin:        repo,
					Owner:         r.Owner.UserName,
					Hoster:        types.GetHost(repo.URL),
				})
				if r.HasWiki && repo.Wiki && types.StatRemote(r.CloneURL, r.SSHURL, repo) {
					repos = append(repos, types.Repo{
						Name:          r.Name + ".wiki",
						URL:           types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"),
						SSHURL:        types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"),
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Owner.UserName,
						Hoster:        types.GetHost(repo.URL),
					})
				}

				continue
			}

			if exclude[r.Name] {
				continue
			}

			if len(repo.Include) == 0 {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.CloneURL,
					SSHURL:        r.SSHURL,
					Token:         token,
					Defaultbranch: r.DefaultBranch,
					Origin:        repo,
					Owner:         r.Owner.UserName,
					Hoster:        types.GetHost(repo.URL),
				})
				if r.HasWiki && repo.Wiki && types.StatRemote(r.CloneURL, r.SSHURL, repo) {
					repos = append(repos, types.Repo{
						Name:          r.Name + ".wiki",
						URL:           types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"),
						SSHURL:        types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"),
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Owner.UserName,
						Hoster:        types.GetHost(repo.URL),
					})
				}
			}
		}
		orgopt := gitea.ListOptions{Page: 1, PageSize: 50}
		orgs := []*gitea.Organization{}
		for {
			o, _, err := client.ListUserOrgs(repo.User, gitea.ListOrgsOptions{ListOptions: orgopt})
			if err != nil {
				log.Fatal().
					Str("stage", "gitea").
					Str("url", repo.URL).
					Msg(err.Error())
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
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.CloneURL,
					SSHURL:        r.SSHURL,
					Token:         token,
					Defaultbranch: r.DefaultBranch,
					Origin:        repo,
					Owner:         r.Owner.UserName,
					Hoster:        types.GetHost(repo.URL),
				})
				if r.HasWiki && repo.Wiki && types.StatRemote(r.CloneURL, r.SSHURL, repo) {
					repos = append(repos, types.Repo{
						Name:          r.Name + ".wiki",
						URL:           types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"),
						SSHURL:        types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"),
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Owner.UserName,
						Hoster:        types.GetHost(repo.URL),
					})
				}

				continue
			}

			if exclude[r.Name] {
				continue
			}

			if len(repo.Include) == 0 {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.CloneURL,
					SSHURL:        r.SSHURL,
					Token:         token,
					Defaultbranch: r.DefaultBranch,
					Origin:        repo,
					Owner:         r.Owner.UserName,
					Hoster:        types.GetHost(repo.URL),
				})
				if r.HasWiki && repo.Wiki && types.StatRemote(r.CloneURL, r.SSHURL, repo) {
					repos = append(repos, types.Repo{
						Name:          r.Name + ".wiki",
						URL:           types.DotGitRx.ReplaceAllString(r.CloneURL, ".wiki.git"),
						SSHURL:        types.DotGitRx.ReplaceAllString(r.SSHURL, ".wiki.git"),
						Token:         token,
						Defaultbranch: r.DefaultBranch,
						Origin:        repo,
						Owner:         r.Owner.UserName,
						Hoster:        types.GetHost(repo.URL),
					})
				}
			}
		}
	}

	return repos
}

func getOrgRepos(client *gitea.Client, org *gitea.Organization,
	orgopt gitea.ListOptions, repo types.GenRepo,
) []*gitea.Repository {
	o, _, err := client.ListOrgRepos(org.UserName,
		gitea.ListOrgReposOptions{orgopt})
	if err != nil {
		log.Fatal().Str("stage", "gitea").Str("url", repo.URL).Msg(err.Error())
	}

	return o
}

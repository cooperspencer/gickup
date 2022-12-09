package gogs

import (
	"github.com/cooperspencer/gickup/types"
	"github.com/gogs/go-gogs-client"
	"github.com/rs/zerolog/log"
)

func getRepoVisibility(visibility string) bool {
	switch visibility {
	case "public":
		return false
	case "private":
		return true
	default:
		return true
	}
}

// Backup TODO.
func Backup(r types.Repo, d types.GenRepo, dry bool) {
	repovisibility := getRepoVisibility(d.Visibility.Repositories)
	log.Info().
		Str("stage", "gogs").
		Str("url", d.URL).
		Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

	gogsclient := gogs.NewClient(d.URL, d.GetToken())

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		log.Fatal().
			Str("stage", "gogs").
			Str("url", d.URL).
			Msg(err.Error())
	}

	if d.User == "" && d.CreateOrg {
		d.User = r.Owner
	}

	if d.User != "" {
		user, err = gogsclient.GetUserInfo(d.User)
		if err != nil {
			if d.CreateOrg {
				org, err := gogsclient.CreateOrg(gogs.CreateOrgOption{
					UserName: d.User,
				})
				if err != nil {
					log.Fatal().
						Str("stage", "gogs").
						Str("url", d.URL).
						Msg(err.Error())
				}
				user.ID = org.ID
				user.UserName = org.UserName
			} else {
				log.Fatal().
					Str("stage", "gogs").
					Str("url", d.URL).
					Msg(err.Error())
			}
		}
	}

	if dry {
		return
	}

	repo, err := gogsclient.GetRepo(user.UserName, r.Name)
	if err != nil {
		opts := gogs.MigrateRepoOption{
			RepoName:     r.Name,
			UID:          int(user.ID),
			Mirror:       true,
			CloneAddr:    r.URL,
			AuthUsername: r.Token,
			Private:      repovisibility,
		}

		if r.Token == "" {
			opts = gogs.MigrateRepoOption{
				RepoName:     r.Name,
				UID:          int(user.ID),
				Mirror:       true,
				CloneAddr:    r.URL,
				AuthUsername: r.Origin.User,
				AuthPassword: r.Origin.Password,
				Private:      repovisibility,
			}
		}

		_, err := gogsclient.MigrateRepo(opts)
		if err != nil {
			log.Fatal().
				Str("stage", "gogs").
				Str("url", d.URL).
				Msg(err.Error())
		}

		return
	}

	if repo.Mirror {
		log.Info().
			Str("stage", "gogs").
			Str("url", d.URL).
			Msgf("mirror of %s already exists, syncing instead", types.Blue(r.Name))

		err := gogsclient.MirrorSync(user.UserName, repo.Name)
		if err != nil {
			log.Fatal().
				Str("stage", "gogs").
				Str("url", d.URL).
				Msg(err.Error())
		}

		log.Info().
			Str("stage", "gogs").
			Str("url", d.URL).
			Msgf("successfully synced %s.", types.Blue(r.Name))
	}
}

// Get TODO.
func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gogs {
		log.Info().
			Str("stage", "gogs").
			Str("url", repo.URL).
			Msgf("grabbing repositories from %s", repo.User)

		token := repo.GetToken()
		client := gogs.NewClient(repo.URL, token)
		gogsrepos, err := client.ListUserRepos(repo.User)
		if err != nil {
			log.Fatal().
				Str("stage", "gogs").
				Str("url", repo.URL).
				Msg(err.Error())
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range gogsrepos {
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
				if repo.Wiki {
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

			if len(include) == 0 {
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
				if repo.Wiki {
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
		orgs, err := client.ListUserOrgs(repo.User)
		if err != nil {
			log.Fatal().
				Str("stage", "gogs").
				Str("url", repo.URL).
				Msg(err.Error())
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
							log.Fatal().
								Str("stage", "gogs").
								Str("url", repo.URL).
								Msg(err.Error())
						}

						if len(o) == 0 {
							break
						}

						orgrepos = append(orgrepos, o...)
					}
				} else {
					o, err := client.ListOrgRepos(org.UserName)
					if err != nil {
						log.Fatal().
							Str("stage", "gogs").
							Str("url", repo.URL).
							Msg(err.Error())
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
				if repo.Wiki {
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

				if repo.Wiki {
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

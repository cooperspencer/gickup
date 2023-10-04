package gogs

import (
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/gogs/go-gogs-client"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
)

func getRepoVisibility(visibility string, private bool) bool {
	switch visibility {
	case "public":
		return false
	case "private":
		return true
	default:
		return private
	}
}

// Backup TODO.
func Backup(r types.Repo, d types.GenRepo, dry bool) bool {
	repovisibility := getRepoVisibility(d.Visibility.Repositories, r.Private)
	sub = logger.CreateSubLogger("stage", "gogs", "url", d.URL)
	sub.Info().
		Msgf("mirroring %s to %s", types.Blue(r.Name), d.URL)

	gogsclient := gogs.NewClient(d.URL, d.GetToken())

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		sub.Error().
			Msg(err.Error())
		return false
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
					sub.Error().
						Msg(err.Error())
					return false
				}
				user.ID = org.ID
				user.UserName = org.UserName
			} else {
				sub.Error().
					Msg(err.Error())
				return false
			}
		}
	}

	if dry {
		return true
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
			Description:  r.Description,
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
				Description:  r.Description,
			}
		}

		_, err := gogsclient.MigrateRepo(opts)
		if err != nil {
			sub.Error().
				Msg(err.Error())
			sub.Info().
				Msgf("deleting %s again", types.Blue(r.Name))
			err = gogsclient.DeleteRepo(user.UserName, r.Name)
			if err != nil {
				sub.Error().
					Msgf("couldn't delete %s!", types.Red(r.Name))
			}
			return false
		}

		return true
	}

	if repo.Mirror {
		sub.Info().
			Msgf("mirror of %s already exists, syncing instead", types.Blue(r.Name))

		err := gogsclient.MirrorSync(user.UserName, repo.Name)
		if err != nil {
			sub.Error().
				Msg(err.Error())
			return false
		}

		sub.Info().
			Msgf("successfully synced %s.", types.Blue(r.Name))
	}

	return true
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Gogs {
		sub = logger.CreateSubLogger("stage", "gogs", "url", repo.URL)
		err := repo.Filter.ParseDuration()
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}
		ran = true
		if repo.User == "" {
			sub.Info().
				Msg("grabbing my repositories")
		} else {
			sub.Info().
				Msgf("grabbing repositories from %s", repo.User)
		}

		token := repo.GetToken()
		client := gogs.NewClient(repo.URL, token)
		var gogsrepos []*gogs.Repository

		if repo.User == "" {
			gogsrepos, err = client.ListMyRepos()
		} else {
			gogsrepos, err = client.ListUserRepos(repo.User)
		}
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		include := types.GetMap(repo.Include)
		includeorgs := types.GetMap(repo.IncludeOrgs)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		for _, r := range gogsrepos {
			if repo.Filter.ExcludeForks {
				if r.Fork {
					continue
				}
			}
			if r.Stars < repo.Filter.Stars {
				continue
			}
			if time.Since(r.Updated) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
				continue
			}
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
					Description:   r.Description,
					Private:       r.Private,
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
						Description:   r.Description,
						Private:       r.Private,
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
					Description:   r.Description,
					Private:       r.Private,
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
						Description:   r.Description,
						Private:       r.Private,
					})
				}
			}
		}

		var orgs []*gogs.Organization

		if repo.User == "" {
			orgs, err = client.ListMyOrgs()
		} else {
			orgs, err = client.ListUserOrgs(repo.User)
		}
		if err != nil {
			sub.Error().
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
							sub.Error().
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
						sub.Error().
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
			if repo.Filter.ExcludeForks {
				if r.Fork {
					continue
				}
			}
			if r.Stars < repo.Filter.Stars {
				continue
			}
			if time.Since(r.Updated) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
				continue
			}

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
					Description:   r.Description,
					Private:       r.Private,
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
						Description:   r.Description,
						Private:       r.Private,
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
					Description:   r.Description,
					Private:       r.Private,
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
						Description:   r.Description,
						Private:       r.Private,
					})
				}
			}
		}
	}

	return repos, ran
}

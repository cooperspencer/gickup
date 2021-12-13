package gitea

import (
	"gickup/types"

	"code.gitea.io/sdk/gitea"
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
	giteaclient.SetBasicAuth(d.Token, "")
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
		i := 0
		gitearepos := []*gitea.Repository{}
		for {
			opt.Page = i
			client, err := gitea.NewClient(repo.Url)
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if repo.Token != "" {
				client.SetBasicAuth(repo.Token, "")
			}
			repos, _, err := client.ListUserRepos(repo.User, opt)
			if err != nil {
				log.Fatal().Str("stage", "gitea").Str("url", repo.Url).Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			gitearepos = append(gitearepos, repos...)
			i++
		}

		exclude := types.GetExcludedMap(repo.Exclude)

		for _, r := range gitearepos {
			if exclude[r.Name] {
				continue
			}
			repos = append(repos, types.Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo})
		}
	}
	return repos
}

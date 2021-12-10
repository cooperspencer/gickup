package main

import (
	"github.com/gogs/go-gogs-client"
	"github.com/rs/zerolog/log"
)

func BackupGogs(r Repo, d GenRepo) {
	log.Info().Str("stage", "gogs").Str("url", d.Url).Msgf("mirroring %s to %s", blue(r.Name), d.Url)
	gogsclient := gogs.NewClient(d.Url, d.Token)

	user, err := gogsclient.GetSelfInfo()
	if err != nil {
		log.Panic().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
	}
	if !dry {
		_, err = gogsclient.GetRepo(user.UserName, r.Name)
		if err != nil {
			opts := gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Token}
			if r.Token == "" {
				opts = gogs.MigrateRepoOption{RepoName: r.Name, UID: int(user.ID), Mirror: true, CloneAddr: r.Url, AuthUsername: r.Origin.User, AuthPassword: r.Origin.Password}
			}
			_, err := gogsclient.MigrateRepo(opts)
			if err != nil {
				log.Panic().Str("stage", "gogs").Str("url", d.Url).Msg(err.Error())
			}
		}
	}
}

func getGogs(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Gogs {
		log.Info().Str("stage", "gogs").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)
		client := gogs.NewClient(repo.Url, repo.Token)
		gogsrepos, err := client.ListUserRepos(repo.User)
		if err != nil {
			log.Panic().Str("stage", "gogs").Str("url", repo.Url).Msg(err.Error())
		}

		exclude := GetExcludedMap(repo.Exclude)

		for _, r := range gogsrepos {
			if exclude[r.Name] {
				continue
			}
			repos = append(repos, Repo{Name: r.Name, Url: r.CloneURL, SshUrl: r.SSHURL, Token: repo.Token, Defaultbranch: r.DefaultBranch, Origin: repo})
		}
	}
	return repos
}

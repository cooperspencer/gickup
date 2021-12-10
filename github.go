package main

import (
	"context"

	"github.com/google/go-github/v41/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

func getGithub(conf *Conf) []Repo {
	repos := []Repo{}
	for _, repo := range conf.Source.Github {
		log.Info().Str("stage", "github").Str("url", "https://github.com").Msgf("grabbing the repositories from %s", repo.User)
		client := &github.Client{}
		opt := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 50}}
		i := 1
		githubrepos := []*github.Repository{}
		if repo.Token == "" {
			client = github.NewClient(nil)
		} else {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: repo.Token},
			)
			tc := oauth2.NewClient(context.TODO(), ts)
			client = github.NewClient(tc)
		}
		for {
			opt.Page = i
			repos, _, err := client.Repositories.List(context.TODO(), repo.User, opt)
			if err != nil {
				log.Panic().Str("stage", "github").Str("url", "https://github.com").Msg(err.Error())
			}
			if len(repos) == 0 {
				break
			}
			githubrepos = append(githubrepos, repos...)
			i++
		}

		exclude := GetExcludedMap(repo.Exclude)
		excludeorgs := GetExcludedMap(repo.ExcludeOrgs)

		for _, r := range githubrepos {
			if exclude[*r.Name] {
				continue
			}
			if excludeorgs[r.GetOwner().GetLogin()] {
				continue
			}

			repos = append(repos, Repo{Name: r.GetName(), Url: r.GetCloneURL(), SshUrl: r.GetSSHURL(), Token: repo.Token, Defaultbranch: r.GetDefaultBranch(), Origin: repo})
		}
	}
	return repos
}

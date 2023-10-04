package bitbucket

import (
	"net/url"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
)

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.BitBucket {
		ran = true
		client := bitbucket.NewBasicAuth(repo.Username, repo.Password)
		if repo.User == "" {
			repo.User = repo.Username
		}

		if repo.URL == "" {
			repo.URL = bitbucket.DEFAULT_BITBUCKET_API_BASE_URL
			sub = logger.CreateSubLogger("stage", "bitbucket", "url", repo.URL)
		} else {
			bitbucketURL, err := url.Parse(repo.URL)
			sub = logger.CreateSubLogger("stage", "bitbucket", "url", repo.URL)
			if err != nil {
				sub.Error().
					Msg(err.Error())
				continue
			}
			client.SetApiBaseURL(*bitbucketURL)
		}

		err := repo.Filter.ParseDuration()
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}

		sub.Info().
			Msgf("grabbing repositories from %s", repo.User)

		repositories, err := client.Repositories.ListForAccount(&bitbucket.RepositoriesOptions{Owner: repo.User})
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)

		for _, r := range repositories.Items {
			user := repo.User
			if r.Owner != nil {
				if _, ok := r.Owner["nickname"]; ok {
					user = r.Owner["nickname"].(string)
				}
			}

			if time.Since(*r.UpdatedOnTime) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
				continue
			}

			if include[r.Name] {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.Links["clone"].([]interface{})[0].(map[string]interface{})["href"].(string),
					SSHURL:        r.Links["clone"].([]interface{})[1].(map[string]interface{})["href"].(string),
					Token:         "",
					Defaultbranch: r.Mainbranch.Name,
					Origin:        repo,
					Owner:         user,
					Hoster:        types.GetHost(repo.URL),
					Description:   r.Description,
					Private:       r.Is_private,
				})

				continue
			}

			if exclude[r.Name] {
				continue
			}

			if len(include) == 0 {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           r.Links["clone"].([]interface{})[0].(map[string]interface{})["href"].(string),
					SSHURL:        r.Links["clone"].([]interface{})[1].(map[string]interface{})["href"].(string),
					Token:         "",
					Defaultbranch: r.Mainbranch.Name,
					Origin:        repo,
					Owner:         user,
					Hoster:        types.GetHost(repo.URL),
					Description:   r.Description,
					Private:       r.Is_private,
				})
			}
		}
	}

	return repos, ran
}

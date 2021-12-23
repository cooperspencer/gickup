package bitbucket

import (
	"gickup/types"
	"net/url"

	"github.com/ktrysmt/go-bitbucket"
	"github.com/rs/zerolog/log"
)

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.BitBucket {
		client := bitbucket.NewBasicAuth(repo.Username, repo.Password)
		if repo.Url == "" {
			repo.Url = bitbucket.DEFAULT_BITBUCKET_API_BASE_URL
		} else {
			bitbucketUrl, err := url.Parse(repo.Url)
			if err != nil {
				log.Fatal().Str("stage", "bitbucket").Str("url", repo.Url).Msg(err.Error())
			}
			client.SetApiBaseURL(*bitbucketUrl)
		}
		log.Info().Str("stage", "bitbucket").Str("url", repo.Url).Msgf("grabbing repositories from %s", repo.User)

		repositories, err := client.Repositories.ListForAccount(&bitbucket.RepositoriesOptions{Owner: repo.User})
		if err != nil {
			log.Fatal().Str("stage", "bitbucket").Str("url", repo.Url).Msg(err.Error())
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
			if include[r.Name] {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.Links["clone"].([]interface{})[0].(map[string]interface{})["href"].(string), SshUrl: r.Links["clone"].([]interface{})[1].(map[string]interface{})["href"].(string), Token: "", Defaultbranch: r.Mainbranch.Name, Origin: repo, Owner: user, Hoster: types.GetHost(repo.Url)})
				continue
			}
			if exclude[r.Name] {
				continue
			}
			if len(include) == 0 {
				repos = append(repos, types.Repo{Name: r.Name, Url: r.Links["clone"].([]interface{})[0].(map[string]interface{})["href"].(string), SshUrl: r.Links["clone"].([]interface{})[1].(map[string]interface{})["href"].(string), Token: "", Defaultbranch: r.Mainbranch.Name, Origin: repo, Owner: user, Hoster: types.GetHost(repo.Url)})
			}
		}
	}
	return repos
}

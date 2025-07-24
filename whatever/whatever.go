package whatever

import (
	"os"
	"strings"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	if len(conf.Source.Any) > 0 {
		ran = true
		log.Info().
			Str("stage", "whatever").
			Msgf("adding repos")
		for _, repo := range conf.Source.Any {
			if repo.URL == "" {
				log.Error().
					Str("stage", "whatever").
					Msg("no url configured")
			}

			hoster := "local"
			if repo.User == "" {
				if repo.Username != "" {
					repo.User = repo.Username
				} else {
					repo.User = "git"
				}
			}
			if _, err := os.Stat(repo.URL); os.IsNotExist(err) {
				hoster = types.GetHost(repo.URL)
			}

			separator := "/"
			if hoster == "local" {
				separator = string(os.PathSeparator)
			}
			name := repo.URL[strings.LastIndex(repo.URL, separator)+1:]
			if strings.HasSuffix(name, ".git") {
				name = name[:strings.LastIndex(name, ".git")]
			}

			repos = append(repos, types.Repo{
				Name:   name,
				URL:    repo.URL,
				SSHURL: repo.URL,
				Token:  repo.GetToken(),
				Origin: repo,
				Owner:  repo.User,
				Hoster: hoster,
			})
		}
	}
	return repos, ran
}

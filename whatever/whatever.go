package whatever

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/rs/zerolog/log"
)

func getRepoNameAndHost(rawURL string) (string, string) {
	endpoint, err := transport.NewEndpoint(rawURL)
	if err == nil {
		if endpoint.Protocol == "file" {
			name := filepath.Base(filepath.Clean(endpoint.Path))
			return strings.TrimSuffix(name, ".git"), "local"
		}

		repoPath := endpoint.Path
		if index := strings.IndexAny(repoPath, "?#"); index >= 0 {
			repoPath = repoPath[:index]
		}
		name := path.Base(strings.TrimRight(repoPath, "/"))
		return strings.TrimSuffix(name, ".git"), endpoint.Host
	}

	name := filepath.Base(filepath.Clean(rawURL))
	return strings.TrimSuffix(name, ".git"), "local"
}

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
				continue
			}

			if repo.User == "" {
				if repo.Username != "" {
					repo.User = repo.Username
				} else {
					repo.User = "git"
				}
			}
			name, hoster := getRepoNameAndHost(repo.URL)

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

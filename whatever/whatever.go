package whatever

import (
	"os"
	"path"
	"strings"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rs/zerolog/log"
)

// Get TODO.
func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	for _, repo := range conf.Source.Any {
		log.Info().
			Str("stage", "whatever").
			Msgf("adding repos")
		if repo.URL == "" {
			log.Error().
				Str("stage", "whatever").
				Msg("no url configured")
		}

		var auth transport.AuthMethod
		hoster := "local"
		if _, err := os.Stat(repo.URL); os.IsNotExist(err) {
			hoster = types.GetHost(repo.URL)
			if strings.HasPrefix(repo.URL, "http://") || strings.HasPrefix(repo.URL, "https://") {
				if repo.Token != "" {
					auth = &http.BasicAuth{
						Username: "xyz",
						Password: repo.Token,
					}
				} else if repo.Username != "" && repo.Password != "" {
					auth = &http.BasicAuth{
						Username: repo.Username,
						Password: repo.Password,
					}
				}
			} else {
				var err error
				if repo.SSHKey == "" {
					home := os.Getenv("HOME")
					repo.SSHKey = path.Join(home, ".ssh", "id_rsa")
				}
				auth, err = ssh.NewPublicKeysFromFile("git", repo.SSHKey, "")
				if err != nil {
					log.Error().
						Str("stage", "whatever").
						Err(err)
					continue
				}
			}
		}

		rem := git.NewRemote(nil, &config.RemoteConfig{Name: "origin", URLs: []string{repo.URL}})
		data, err := rem.List(&git.ListOptions{Auth: auth})
		if err != nil {
			log.Error().
				Str("stage", "whatever").
				Err(err)

			continue
		}

		main := ""
		for _, d := range data {
			if d.Hash().IsZero() {
				main = d.Target().Short()

				break
			}
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
			Name:          name,
			URL:           repo.URL,
			SSHURL:        repo.URL,
			Token:         repo.GetToken(),
			Defaultbranch: main,
			Origin:        repo,
			Owner:         "git",
			Hoster:        hoster,
		})
	}

	return repos
}

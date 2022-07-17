package any

import (
	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rs/zerolog/log"
	"os"
	"path"
	"strings"
)

// Get TODO.
func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}
	log.Info().
		Str("stage", "any").
		Msgf("adding repos")
	for _, repo := range conf.Source.Any {
		if repo.URL == "" {
			log.Error().
				Str("stage", "any").
				Msg("no url configured")
		}

		var auth transport.AuthMethod
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
					Str("stage", "any").
					Err(err)
				continue
			}
		}

		rem := git.NewRemote(nil, &config.RemoteConfig{Name: "origin", URLs: []string{repo.URL}})
		data, err := rem.List(&git.ListOptions{Auth: auth})
		if err != nil {
			log.Error().
				Str("stage", "any").
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

		name := repo.URL[strings.LastIndex(repo.URL, "/")+1:]
		if strings.HasSuffix(name, ".git") {
			name = name[strings.LastIndex(name, ".git")+1:]
		}

		repos = append(repos, types.Repo{
			Name:          name,
			URL:           repo.URL,
			SSHURL:        repo.URL,
			Token:         repo.GetToken(),
			Defaultbranch: main,
			Origin:        repo,
			Owner:         "git",
			Hoster:        types.GetHost(repo.URL),
		})

	}

	return repos
}

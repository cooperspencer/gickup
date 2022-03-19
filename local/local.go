package local

import (
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/melbahja/goph"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
)

func Locally(repo types.Repo, l types.Local, dry bool) {
	if l.Structured {
		repo.Name = path.Join(repo.Hoster, repo.Owner, repo.Name)
	}
	stat, err := os.Stat(l.Path)
	if os.IsNotExist(err) && !dry {
		err := os.MkdirAll(l.Path, 0o777)
		if err != nil {
			log.Fatal().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
		}
		stat, _ = os.Stat(l.Path)
	}
	if stat != nil {
		if stat.IsDir() {
			os.Chdir(l.Path)
		}
	}

	tries := 5
	var auth transport.AuthMethod
	if repo.Origin.SSH {
		if repo.Origin.SSHKey == "" {
			home := os.Getenv("HOME")
			repo.Origin.SSHKey = path.Join(home, ".ssh", "id_rsa")
		}
		auth, err = ssh.NewPublicKeysFromFile("git", repo.Origin.SSHKey, "")
		if err != nil {
			panic(err)
		}
	} else if repo.Token != "" {
		auth = &http.BasicAuth{
			Username: "xyz",
			Password: repo.Token,
		}
	} else if repo.Origin.Username != "" && repo.Origin.Password != "" {
		auth = &http.BasicAuth{
			Username: repo.Origin.Username,
			Password: repo.Origin.Password,
		}
	}
	for x := 1; x <= tries; x++ {
		stat, err := os.Stat(repo.Name)
		if os.IsNotExist(err) {
			log.Info().Str("stage", "locally").Str("path", l.Path).Msgf("cloning %s", types.Green(repo.Name))

			err := cloneRepository(repo, auth, dry)
			if err != nil {
				if x == tries {
					log.Fatal().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msg(err.Error())
				} else {
					if strings.Contains(err.Error(), "ERR access denied or repository not exported") {
						log.Warn().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msgf("%s doesn't exist.", repo.Name)
						break
					}
					if strings.Contains(err.Error(), "remote repository is empty") {
						log.Warn().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msg(err.Error())
						break
					}
					log.Warn().Str("stage", "locally").Str("path", l.Path).Msgf("retry %s from %s", types.Red(x), types.Red(tries))
					time.Sleep(5 * time.Second)
					continue
				}
			}
		} else {
			if !stat.IsDir() {
				log.Warn().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msgf("%s is a file", types.Red(repo.Name))
			} else {
				log.Info().Str("stage", "locally").Str("path", l.Path).Msgf("opening %s locally", types.Green(repo.Name))

				err := updateRepository(repo.Name, auth, dry)
				if err != nil {
					if strings.Contains(err.Error(), "already up-to-date") {
						log.Info().Str("stage", "locally").Str("path", l.Path).Msg(err.Error())
					} else {
						if x == tries {
							log.Fatal().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msg(err.Error())
						} else {
							os.RemoveAll(repo.Name)
							log.Warn().Str("stage", "locally").Str("path", l.Path).Str("repo", repo.Name).Msgf("retry %s from %s", types.Red(x), types.Red(tries))
							time.Sleep(5 * time.Second)
							continue
						}
					}
				}
			}
		}
		x = 5
	}
}

func updateRepository(repoPath string, auth transport.AuthMethod, dry bool) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if !dry {
		log.Info().Str("stage", "locally").Msgf("pulling %s", types.Green(repoPath))
		err = w.Pull(&git.PullOptions{Auth: auth, RemoteName: "origin", SingleBranch: false})
	}
	return err
}

func cloneRepository(repo types.Repo, auth transport.AuthMethod, dry bool) error {
	if !dry {
		url := repo.Url
		if repo.Origin.SSH {
			url = repo.SshUrl
			site := types.Site{}
			err := site.GetValues(url)
			if err != nil {
				log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
			}

			sshAuth, err := goph.Key(repo.Origin.SSHKey, "")
			if err != nil {
				log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
			}

			err = testSshConnection(site, sshAuth)
			if err != nil {
				log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
			}
		}

		rem := git.NewRemote(nil, &config.RemoteConfig{Name: "origin", URLs: []string{url}})
		_, err := rem.List(&git.ListOptions{Auth: auth})
		if err != nil {
			return err
		}

		_, err = git.PlainClone(repo.Name, false, &git.CloneOptions{
			URL:          url,
			Auth:         auth,
			SingleBranch: false,
		})

		return err
	}
	return nil
}

func testSshConnection(site types.Site, sshAuth goph.Auth) error {
	_, err := goph.NewConn(&goph.Config{
		User:     site.User,
		Addr:     site.Url,
		Port:     uint(site.Port),
		Auth:     sshAuth,
		Callback: VerifyHost,
	})

	return err
}

func VerifyHost(host string, remote net.Addr, key gossh.PublicKey) error {
	// Got from the example from https://github.com/melbahja/goph/blob/master/examples/goph/main.go
	//
	// If you want to connect to new hosts.
	// here your should check new connections public keys
	// if the key not trusted you shuld return an error
	//

	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file OR host in known hosts file but key changed!
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")
	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	/*
		if hostFound && err != nil {
			return err
		}
	*/
	// handshake because public key already exists.
	if hostFound && err == nil {
		return nil
	}

	// Add the new host to known hosts file.
	return goph.AddKnownHost(host, remote, key, "")
}

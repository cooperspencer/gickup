package local

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/melbahja/goph"
	"github.com/mholt/archiver"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
)

// Locally TODO.
func Locally(repo types.Repo, l types.Local, dry bool) {
	date := time.Now()
	search := fmt.Sprintf("(?m)%s_[0-9]{10}", repo.Name)
	if l.Zip {
		search += ".zip"
	}
	if l.Structured {
		if l.Date {
			repo.Name = path.Join(fmt.Sprint(date.Year()), fmt.Sprintf("%02d", int(date.Month())), fmt.Sprintf("%02d", date.Day()), repo.Hoster, repo.Owner, repo.Name)
		} else {
			repo.Name = path.Join(repo.Hoster, repo.Owner, repo.Name)
		}
	}

	if l.Keep > 0 {
		repo.Name += fmt.Sprintf("_%d", date.Unix())
	}

	stat, err := os.Stat(l.Path)
	if os.IsNotExist(err) && !dry {
		if err := os.MkdirAll(l.Path, 0o777); err != nil {
			log.Fatal().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
		}

		stat, _ = os.Stat(l.Path)
	}

	if stat != nil && stat.IsDir() {
		if err := os.Chdir(l.Path); err != nil {
			log.Fatal().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
		}
	}

	tries := 5

	var auth transport.AuthMethod

	switch {
	case repo.Origin.SSH:
		if repo.Origin.SSHKey == "" {
			home := os.Getenv("HOME")
			repo.Origin.SSHKey = path.Join(home, ".ssh", "id_rsa")
		}

		auth, err = ssh.NewPublicKeysFromFile("git", repo.Origin.SSHKey, "")
		if err != nil {
			panic(err)
		}
	case repo.Token != "":
		auth = &http.BasicAuth{
			Username: "xyz",
			Password: repo.Token,
		}
	case repo.Origin.Username != "" && repo.Origin.Password != "":
		auth = &http.BasicAuth{
			Username: repo.Origin.Username,
			Password: repo.Origin.Password,
		}
	}

	for x := 1; x <= tries; x++ {
		stat, err := os.Stat(repo.Name)
		if os.IsNotExist(err) {
			log.Info().
				Str("stage", "locally").
				Str("path", l.Path).
				Msgf("cloning %s", types.Green(repo.Name))

			err := cloneRepository(repo, auth, dry)
			if err != nil {
				if err.Error() == "repository not found" {
					log.Warn().
						Str("stage", "locally").
						Str("path", l.Path).
						Str("repo", repo.Name).
						Msg(err.Error())
					break
				}
				if x == tries {
					log.Warn().
						Str("stage", "locally").
						Str("path", l.Path).
						Str("repo", repo.Name).
						Msg(err.Error())

					break
				}

				if strings.Contains(err.Error(), "ERR access denied or repository not exported") {
					log.Warn().
						Str("stage", "locally").
						Str("path", l.Path).
						Str("repo", repo.Name).
						Msgf("%s doesn't exist.", repo.Name)

					break
				}

				if strings.Contains(err.Error(), "remote repository is empty") {
					log.Warn().
						Str("stage", "locally").
						Str("path", l.Path).
						Str("repo", repo.Name).
						Msg(err.Error())

					break
				}

				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Msgf("retry %s from %s", types.Red(x), types.Red(tries))

				time.Sleep(5 * time.Second)

				continue
			}
		} else {
			if !stat.IsDir() {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).
					Msgf("%s is a file", types.Red(repo.Name))
			} else {
				log.Info().
					Str("stage", "locally").
					Str("path", l.Path).
					Msgf("opening %s locally", types.Green(repo.Name))

				err := updateRepository(repo.Name, auth, dry)
				if err != nil {
					if strings.Contains(err.Error(), "already up-to-date") {
						log.Info().
							Str("stage", "locally").
							Str("path", l.Path).
							Msg(err.Error())
					} else {
						if x == tries {
							log.Fatal().
								Str("stage", "locally").
								Str("path", l.Path).
								Str("repo", repo.Name).
								Msg(err.Error())
						} else {
							os.RemoveAll(repo.Name)
							log.Warn().
								Str("stage", "locally").
								Str("path", l.Path).
								Str("repo", repo.Name).
								Msgf("retry %s from %s", types.Red(x), types.Red(tries))

							time.Sleep(5 * time.Second)

							continue
						}
					}
				}
			}
		}

		if l.Zip {
			log.Info().
				Str("stage", "locally").
				Str("path", l.Path).
				Msgf("zipping %s", types.Green(repo.Name))
			err := archiver.Archive([]string{repo.Name}, fmt.Sprintf("%s.zip", repo.Name))
			if err != nil {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).Err(err)
			}
			err = os.RemoveAll(repo.Name)
			if err != nil {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).Err(err)
			}
		}

		if l.Keep > 0 {
			var re = regexp.MustCompile(search)

			parentdir := path.Dir(repo.Name)
			files, err := ioutil.ReadDir(parentdir)
			if err != nil {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).Err(err)
				break
			}

			keep := []string{}
			for _, file := range files {
				match := re.FindAllString(file.Name(), -1)
				if len(match) > 0 {
					if !l.Zip {
						if strings.HasSuffix(file.Name(), ".zip") {
							continue
						}
					}
					keep = append(keep, file.Name())
				}
			}

			sort.Sort(sort.Reverse(sort.StringSlice(keep)))

			if len(keep) > l.Keep {
				toremove := keep[l.Keep:]
				for _, file := range toremove {
					log.Info().
						Str("stage", "locally").
						Str("path", l.Path).
						Msgf("removing %s", types.Red(path.Join(parentdir, file)))
					err := os.RemoveAll(path.Join(parentdir, file))
					if err != nil {
						log.Warn().
							Str("stage", "locally").
							Str("path", l.Path).
							Str("repo", repo.Name).Err(err)
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
		log.Info().
			Str("stage", "locally").
			Msgf("pulling %s", types.Green(repoPath))

		err = w.Pull(&git.PullOptions{Auth: auth, RemoteName: "origin", SingleBranch: false})
	}

	return err
}

func cloneRepository(repo types.Repo, auth transport.AuthMethod, dry bool) error {
	if dry {
		return nil
	}

	url := repo.URL
	if repo.Origin.SSH {
		url = repo.SSHURL
		site := types.Site{}

		err := site.GetValues(url)
		if err != nil {
			log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
		}

		sshAuth, err := goph.Key(repo.Origin.SSHKey, "")
		if err != nil {
			log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
		}

		err = testSSHConnection(site, sshAuth)
		if err != nil {
			log.Fatal().Str("stage", "locally").Str("repo", repo.Name).Msg(err.Error())
		}
	}

	remoteConfig := config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	}

	rem := git.NewRemote(nil, &remoteConfig)

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

func testSSHConnection(site types.Site, sshAuth goph.Auth) error {
	_, err := goph.NewConn(&goph.Config{
		User:     site.User,
		Addr:     site.URL,
		Port:     uint(site.Port),
		Auth:     sshAuth,
		Callback: VerifyHost,
	})

	return err
}

// VerifyHost TODO.
func VerifyHost(host string, remote net.Addr, key gossh.PublicKey) error {
	// Got from the example from
	// https://github.com/melbahja/goph/blob/master/examples/goph/main.go
	//
	// If you want to connect to new hosts.
	// here your should check new connections public keys
	// if the key not trusted you shuld return an error
	//

	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file
	// OR host in known hosts file but key changed!
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

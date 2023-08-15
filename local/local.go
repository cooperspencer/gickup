package local

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/gitcmd"
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

var (
	gitc = gitcmd.GitCmd{}
)

// Locally TODO.
func Locally(repo types.Repo, l types.Local, dry bool) bool {
	if l.LFS {
		g, err := gitcmd.New()
		if err != nil {
			log.Error().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
		}
		gitc = g
	}
	date := time.Now()

	if l.Structured {
		repo.Name = path.Join(repo.Hoster, repo.Owner, repo.Name)
	}

	if l.Bare {
		repo.Name += ".git"
	}

	if l.Keep > 0 {
		repo.Name = path.Join(repo.Name, fmt.Sprint(date.Unix()))
	}

	stat, err := os.Stat(l.Path)
	if os.IsNotExist(err) && !dry {
		if err := os.MkdirAll(l.Path, 0o777); err != nil {
			log.Error().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
			return false
		}

		stat, _ = os.Stat(l.Path)
	}

	if stat != nil && stat.IsDir() {
		if err := os.Chdir(l.Path); err != nil {
			log.Error().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
			return false
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
			log.Error().
				Str("stage", "locally").
				Str("path", l.Path).
				Msg(err.Error())
			return false
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

			err := cloneRepository(repo, auth, dry, l)
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

				err := updateRepository(repo.Name, auth, dry, l)
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
					Str("repo", repo.Name).Msg(err.Error())
			}
			err = os.RemoveAll(repo.Name)
			if err != nil {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).Msg(err.Error())
			}
		}

		if l.Keep > 0 {
			parentdir := path.Dir(repo.Name)
			files, err := os.ReadDir(parentdir)
			if err != nil {
				log.Warn().
					Str("stage", "locally").
					Str("path", l.Path).
					Str("repo", repo.Name).Msg(err.Error())
				break
			}

			keep := []string{}
			for _, file := range files {
				fname := file.Name()
				if l.Zip {
					fname = strings.TrimSuffix(file.Name(), ".zip")
				}
				_, err := strconv.ParseInt(fname, 10, 64)
				if err != nil {
					log.Warn().
						Str("stage", "locally").
						Str("path", l.Path).
						Str("repo", repo.Name).
						Msgf("couldn't parse timestamp! %s", types.Red(file.Name()))
				}
				if l.Zip && !strings.HasSuffix(file.Name(), ".zip") {
					continue
				}
				keep = append(keep, file.Name())
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
							Str("repo", repo.Name).Msg(err.Error())
					}
				}
			}
		}

		x = 5
	}
	return true
}

func updateRepository(repoPath string, auth transport.AuthMethod, dry bool, l types.Local) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	if !dry {
		if l.LFS {
			err = os.Chdir(repoPath)
			if err != nil {
				return err
			}

			log.Info().
				Str("stage", "locally").
				Msgf("pulling %s", types.Green(repoPath))

			err = gitc.Pull(l.Bare)
			if err != nil {
				return err
			}
		} else {
			if l.Bare {
				err = r.Fetch(&git.FetchOptions{Auth: auth, RemoteName: "origin", RefSpecs: []config.RefSpec{"+refs/*:refs/*"}})
			} else {
				w, err := r.Worktree()
				if err != nil {
					return err
				}

				log.Info().
					Str("stage", "locally").
					Msgf("pulling %s", types.Green(repoPath))

				err = w.Pull(&git.PullOptions{Auth: auth, RemoteName: "origin", SingleBranch: false})
			}
		}
	}
	return err
}

func cloneRepository(repo types.Repo, auth transport.AuthMethod, dry bool, l types.Local) error {
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

	if l.LFS {
		err = gitc.Clone(url, repo.Name, l.Bare)
	} else {
		_, err = git.PlainClone(repo.Name, l.Bare, &git.CloneOptions{
			URL:          url,
			Auth:         auth,
			SingleBranch: false,
		})
	}

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

func TempClone(repo types.Repo, tempdir string) (*git.Repository, error) {
	auth := &http.BasicAuth{
		Username: "xyz",
		Password: repo.Token,
	}
	r, err := git.PlainClone(tempdir, false, &git.CloneOptions{
		URL:  repo.URL,
		Auth: auth,
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}

func CreateRemotePush(repo *git.Repository, destination types.GenRepo, url string) error {
	token := destination.GetToken()
	remoteconfig := config.RemoteConfig{Name: RandomString(8), URLs: []string{url}}
	remote, err := repo.CreateRemote(&remoteconfig)
	if err != nil {
		return err
	}

	auth := &http.BasicAuth{
		Username: "xyz",
		Password: token,
	}

	pushoptions := git.PushOptions{Auth: auth, RemoteName: remote.Config().Name}

	return repo.Push(&pushoptions)
}

func RandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

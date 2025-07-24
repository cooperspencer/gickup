package local

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/gitcmd"
	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/cooperspencer/gickup/zip"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/melbahja/goph"
	"github.com/rs/zerolog"
	gossh "golang.org/x/crypto/ssh"
)

var (
	gitc = gitcmd.GitCmd{}
	sub  zerolog.Logger
)

// Locally TODO.
func Locally(repo types.Repo, l types.Local, dry bool) bool {
	sub = logger.CreateSubLogger("stage", "locally", "path", l.Path)
	if l.LFS {
		g, err := gitcmd.New()
		if err != nil {
			sub.Error().
				Msg(err.Error())
		}
		gitc = g
	}
	date := time.Now()

	if l.Structured {
		repo.Name = path.Join(repo.Hoster, repo.Owner, repo.Name)
	}

	if l.Bare || l.Mirror {
		repo.Name += ".git"
	}

	if l.Keep > 0 {
		repo.Name = path.Join(repo.Name, fmt.Sprint(date.Unix()))
	}

	_, err := os.Stat(l.Path)
	if os.IsNotExist(err) && !dry {
		if err = os.MkdirAll(l.Path, 0o777); err != nil {
			sub.Error().
				Msg(err.Error())

			return false
		}

		_, err = os.Stat(l.Path)
	}

	if err != nil {
		sub.Error().
			Msg(err.Error())

		return false
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
			sub.Error().
				Msg(err.Error())

			return false
		}
	case repo.Token != "":
		if repo.NoTokenUser {
			auth = &http.BasicAuth{
				Username: repo.Token,
			}
		} else {
			auth = &http.BasicAuth{
				Username: repo.Origin.User,
				Password: repo.Token,
			}
		}
	case repo.Origin.Username != "" && repo.Origin.Password != "":
		auth = &http.BasicAuth{
			Username: repo.Origin.Username,
			Password: repo.Origin.Password,
		}
	}

	for x := 1; x <= tries; x++ {
		stat, err := os.Stat(filepath.Join(l.Path, repo.Name))
		if os.IsNotExist(err) {
			sub.Info().
				Msgf("cloning %s", types.Green(repo.Name))

			err := cloneRepository(repo, auth, dry, l)
			if err != nil {
				if err.Error() == "repository not found" {
					sub.Warn().
						Str("repo", repo.Name).
						Msg(err.Error())

					return false
				}
				if x == tries {
					sub.Warn().
						Str("repo", repo.Name).
						Msg(err.Error())

					return false
				}

				if strings.Contains(err.Error(), "ERR access denied or repository not exported") {
					sub.Warn().
						Str("repo", repo.Name).
						Msgf("%s doesn't exist.", repo.Name)

					return false
				}

				if strings.Contains(err.Error(), "remote repository is empty") {
					sub.Warn().
						Str("repo", repo.Name).
						Msg(err.Error())

					break
				}

				/*
					err = os.RemoveAll(filepath.Join(l.Path, repo.Name))
					if err != nil {
						dir, _ := filepath.Abs(filepath.Join(l.Path, repo.Name))
						sub.Warn().
							Str("repo", repo.Name).Err(err).
							Msgf("couldn't remove %s", types.Red(dir))
					}
				*/

				sub.Warn().Err(err).
					Msgf("retry %s from %s", types.Red(x), types.Red(tries))

				time.Sleep(5 * time.Second)

				continue
			}
		} else {
			if !stat.IsDir() {
				sub.Warn().
					Str("repo", repo.Name).
					Msgf("%s is a file", types.Red(repo.Name))
			} else {
				sub.Info().
					Msgf("opening %s locally", types.Green(repo.Name))

				err := updateRepository(repo.Name, auth, dry, l)
				if err != nil {
					if err == git.NoErrAlreadyUpToDate {
						sub.Info().
							Msg(err.Error())
					} else {
						if x == tries {
							sub.Warn().
								Str("repo", repo.Name).
								Msg(err.Error())

							return false
						} else {
							sub.Warn().
								Str("repo", repo.Name).Err(err).
								Msgf("retry %s from %s", types.Red(x), types.Red(tries))

							time.Sleep(5 * time.Second)

							continue
						}
					}
				}
			}
		}

		if len(repo.Issues) > 0 {
			_, err := os.Stat(filepath.Join(l.Path, fmt.Sprintf("%s.issues", repo.Name)))
			if os.IsNotExist(err) && !dry {
				if err := os.MkdirAll(filepath.Join(l.Path, fmt.Sprintf("%s.issues", repo.Name)), 0o777); err != nil {
					sub.Error().
						Msg(err.Error())
				}
			}
			issuesDir, err := filepath.Abs(filepath.Join(l.Path, fmt.Sprintf("%s.issues", repo.Name)))
			if err != nil {
				sub.Error().
					Msg(err.Error())
			} else {
				sub.Info().Str("repo", repo.Name).Msg("backing up issues")
				if !dry {
					for k, v := range repo.Issues {
						jsonData, err := json.Marshal(v)
						if err != nil {
							sub.Error().
								Msg(err.Error())
						} else {
							err = os.WriteFile(filepath.Join(issuesDir, fmt.Sprintf("%s.json", k)), jsonData, 0644)
							if err != nil {
								sub.Error().
									Msg(err.Error())
							}
						}
					}
				}
			}
		}

		if l.Zip {
			tozip := []string{filepath.Join(l.Path, repo.Name)}

			if len(repo.Issues) > 0 {
				tozip = append(tozip, filepath.Join(l.Path, fmt.Sprintf("%s.issues", repo.Name)))
			}
			sub.Info().
				Msgf("zipping %s", types.Green(repo.Name))

			if _, err := os.Stat(fmt.Sprintf("%s.zip", filepath.Join(l.Path, repo.Name))); !os.IsNotExist(err) {
				sub.Warn().Str("repo", repo.Name).Msgf("will overwrite %s.zip", filepath.Join(l.Path, repo.Name))
			}

			err := zip.Zip(filepath.Join(l.Path, repo.Name), tozip)
			if err != nil {
				sub.Error().
					Str("repo", repo.Name).
					Msg(err.Error())

				return false
			}

		}

		if l.Keep > 0 {
			parentdir := path.Dir(filepath.Join(l.Path, repo.Name))
			files, err := os.ReadDir(parentdir)
			if err != nil {
				sub.Warn().
					Str("repo", repo.Name).Msg(err.Error())

				return false
			}

			keep := []string{}
			for _, file := range files {
				fname := file.Name()
				if l.Zip {
					fname = strings.TrimSuffix(file.Name(), ".zip")
				}

				fname = strings.TrimSuffix(fname, ".issues")

				_, err := strconv.ParseInt(fname, 10, 64)
				if err != nil {
					sub.Warn().
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
					sub.Info().
						Msgf("removing %s", types.Red(path.Join(parentdir, file)))
					err := os.RemoveAll(path.Join(parentdir, file))
					if err != nil {
						sub.Warn().
							Str("repo", repo.Name).Msg(err.Error())
					}
				}
			}
		}

		x = 5
	}

	return true
}

func updateRepository(reponame string, auth transport.AuthMethod, dry bool, l types.Local) error {
	r, err := git.PlainOpen(filepath.Join(l.Path, reponame))
	if err != nil {
		return err
	}

	if !dry {
		if l.LFS {
			_, err = os.Stat(filepath.Join(l.Path, reponame))
			if err != nil {
				return err
			}

			sub.Info().
				Msgf("pulling %s", types.Green(reponame))

			err = gitc.Pull(l.Bare, l.Mirror, filepath.Join(l.Path, reponame))
			if err != nil {
				return err
			}

			if l.Bare || l.Mirror {
				sub.Info().
					Msgf("fetching lfs files for %s", types.Green(reponame))

				err = gitc.LFSFetch(filepath.Join(l.Path, reponame))
				if err != nil {
					return err
				}
			}
		} else {
			// fetch to see if there are any unpullable commits, for example a force push
			err = r.Fetch(&git.FetchOptions{Auth: auth, RemoteName: "origin"})
			if err != nil {
				if err == git.NoErrAlreadyUpToDate {
					err = nil
				} else {
					return err
				}
			}
			sub.Info().
				Msgf("pulling %s", types.Green(reponame))
			if !l.Bare && !l.Mirror {
				w, err := r.Worktree()
				if err != nil {
					if err == git.NoErrAlreadyUpToDate {
						err = nil
					} else {
						return err
					}
				}
				err = w.Pull(&git.PullOptions{Auth: auth, RemoteName: "origin", SingleBranch: false})
				if err == git.NoErrAlreadyUpToDate {
					err = nil
				} else {
					return err
				}
			}
			// if everything was ok, fetch everything
			err = r.Fetch(&git.FetchOptions{Auth: auth, RemoteName: "origin", RefSpecs: []config.RefSpec{"+refs/*:refs/*"}})
			if err != nil {
				return err
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
			sub.Fatal().Str("repo", repo.Name).Msg(err.Error())
		}

		sshAuth, err := goph.Key(repo.Origin.SSHKey, "")
		if err != nil {
			sub.Fatal().Str("repo", repo.Name).Msg(err.Error())
		}

		err = testSSHConnection(site, sshAuth)
		if err != nil {
			sub.Fatal().Str("repo", repo.Name).Msg(err.Error())
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
		if repo.Token != "" {
			if strings.HasPrefix(url, "http://") {
				url = strings.Replace(url, "http://", fmt.Sprintf("http://xyz:%s@", repo.Token), -1)
			}

			if strings.HasPrefix(url, "https://") {
				url = strings.Replace(url, "https://", fmt.Sprintf("https://xyz:%s@", repo.Token), -1)
			}
		} else {
			if repo.Origin.Username != "" && repo.Origin.Password != "" {
				if strings.HasPrefix(url, "http://") {
					url = strings.Replace(url, "http://", fmt.Sprintf("http://%s:%s@", repo.Origin.Username, repo.Origin.Password), -1)
				}

				if strings.HasPrefix(url, "https://") {
					url = strings.Replace(url, "https://", fmt.Sprintf("https://%s:%s@", repo.Origin.Username, repo.Origin.Password), -1)
				}
			}
		}

		err = gitc.Clone(url, filepath.Join(l.Path, repo.Name), l.Bare, l.Mirror)
		if err != nil {
			return err
		}

		if l.Bare || l.Mirror {
			sub.Info().
				Msgf("fetching lfs files for %s", types.Green(repo.Name))

			err = gitc.LFSFetch(filepath.Join(l.Path, repo.Name))
			if err != nil {
				return err
			}
		}
	} else {
		r := &git.Repository{}
		r, err = git.PlainClone(filepath.Join(l.Path, repo.Name), l.Bare, &git.CloneOptions{
			URL:          url,
			Auth:         auth,
			SingleBranch: false,
			Mirror:       l.Mirror,
		})
		if err != nil {
			return err
		}
		err = r.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
			Auth:     auth,
			Force:    true,
		})
		if err == git.NoErrAlreadyUpToDate {
			err = nil
		}
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
	return tempCloneBase(repo, tempdir, false)
}

func TempCloneBare(repo types.Repo, tempdir string) (*git.Repository, error) {
	return tempCloneBase(repo, tempdir, true)
}

func tempCloneBase(repo types.Repo, tempdir string, isBare bool) (*git.Repository, error) {
	var auth transport.AuthMethod
	if repo.Token != "" {
		if repo.NoTokenUser {
			auth = &http.BasicAuth{
				Username: repo.Token,
			}
		} else {
			auth = &http.BasicAuth{
				Username: repo.Origin.User,
				Password: repo.Token,
			}
		}
	}
	if repo.Origin.LFS {
		g, err := gitcmd.New()
		if err != nil {
			return nil, err
		}
		gitc = g

		if strings.HasPrefix(repo.URL, "http://") {
			repo.URL = strings.Replace(repo.URL, "http://", fmt.Sprintf("http://xyz:%s@", repo.Token), -1)
		}

		if strings.HasPrefix(repo.URL, "https://") {
			repo.URL = strings.Replace(repo.URL, "https://", fmt.Sprintf("https://xyz:%s@", repo.Token), -1)
		}

		err = gitc.Clone(repo.URL, tempdir, isBare, false)
		if err != nil {
			return nil, err
		}

		r, err := git.PlainOpen(tempdir)
		if err != nil {
			return nil, err
		}
		err = r.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
			Auth:     auth,
			Force:    true,
		})
		if err == git.NoErrAlreadyUpToDate {
			return r, nil
		}

		// Get the symbolic reference for HEAD
		headRef, err := r.Head()
		if err != nil {
			return nil, err
		}

		// Retrieve the list of branches
		refs, err := r.Branches()
		if err != nil {
			return nil, err
		}

		// Print the names of branches
		err = refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Name().Short() != headRef.Name().Short() {
				return gitc.Checkout(tempdir, ref.Name().Short())
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		err = gitc.Checkout(tempdir, headRef.Name().Short())
		if err != nil {
			return nil, err
		}

		err = gitc.MirrorPull(tempdir)
		if err != nil {
			return nil, err
		}

		return r, err
	} else {
		r, err := git.PlainClone(tempdir, isBare, &git.CloneOptions{
			URL:          repo.URL,
			Auth:         auth,
			SingleBranch: false,
		})
		if err != nil {
			return nil, err
		}

		err = r.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
			Auth:     auth,
			Force:    true,
		})
		if err == git.NoErrAlreadyUpToDate {
			return r, nil
		} else {
			return r, err
		}
	}
}

func CreateRemotePush(repo *git.Repository, destination types.GenRepo, url string, lfs bool) error {
	sub = logger.CreateSubLogger("stage", "tempclone", "url", url)
	token := destination.GetToken()
	var auth transport.AuthMethod
	if destination.SSH {
		if destination.SSHKey == "" {
			home := os.Getenv("HOME")
			destination.SSHKey = path.Join(home, ".ssh", "id_rsa")
		}
		site := types.Site{}

		err := site.GetValues(url)
		if err != nil {
			sub.Fatal().Msg(err.Error())
		}

		sshAuth, err := goph.Key(destination.SSHKey, "")
		if err != nil {
			sub.Fatal().Msg(err.Error())
		}

		err = testSSHConnection(site, sshAuth)
		if err != nil {
			sub.Fatal().Msg(err.Error())
		}
		if destination.SSHKey == "" {
			home := os.Getenv("HOME")
			destination.SSHKey = path.Join(home, ".ssh", "id_rsa")
		}

		auth, err = ssh.NewPublicKeysFromFile("git", destination.SSHKey, "")
		if err != nil {
			return err
		}
	} else {
		auth = &http.BasicAuth{
			Username: "xyz",
			Password: token,
		}
	}
	if lfs {
		g, err := gitcmd.New()
		if err != nil {
			return err
		}
		gitc = g
		worktree, err := repo.Worktree()
		if err != nil {
			return err
		}

		remote := RandomString(8)

		if destination.SSH {
			err = gitc.NewRemote(remote, url, worktree.Filesystem.Root())
			if err != nil {
				return err
			}

			err = gitc.SSHPush(worktree.Filesystem.Root(), remote, destination.SSHKey)
			if err != nil {
				return err
			}
		} else {
			if strings.HasPrefix(url, "http://") {
				url = strings.Replace(url, "http://", fmt.Sprintf("http://xyz:%s@", token), -1)
			}

			if strings.HasPrefix(url, "https://") {
				url = strings.Replace(url, "https://", fmt.Sprintf("https://xyz:%s@", token), -1)
			}

			err = gitc.NewRemote(remote, url, worktree.Filesystem.Root())
			if err != nil {
				return err
			}

			err = gitc.Push(worktree.Filesystem.Root(), remote)
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		remoteconfig := config.RemoteConfig{Name: RandomString(8), URLs: []string{url}}
		remote, err := repo.CreateRemote(&remoteconfig)
		if err != nil {
			return err
		}

		headref, _ := repo.Head()

		pushoptions := git.PushOptions{Force: destination.Force, Auth: auth, RemoteName: remote.Config().Name, RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", headref.Name(), headref.Name()))}}

		err = repo.Push(&pushoptions)
		if err == nil || err == git.NoErrAlreadyUpToDate {
			pushoptions = git.PushOptions{Force: destination.Force, Auth: auth, RemoteName: remote.Config().Name, RefSpecs: []config.RefSpec{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"}}

			return repo.Push(&pushoptions)
		}
		return err
	}
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

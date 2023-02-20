package sourcehut

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

// doRequest TODO
func doRequest(url, token string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	return body, err
}

// getRepos TODO
func getRepos(url, token string) (Repositories, error) {
	repositories := Repositories{}

	body, err := doRequest(url, token)
	if err != nil {
		return Repositories{}, err
	}

	err = json.Unmarshal(body, &repositories)
	if err != nil {
		return Repositories{}, err
	}

	for {
		if repositories.Next != "" {
			body, err := doRequest(fmt.Sprintf("%s/id=%s", url, repositories.Next), token)
			if err != nil {
				return Repositories{}, err
			}

			r := Repositories{}

			err = json.Unmarshal(body, &r)
			if err != nil {
				return Repositories{}, err
			}

			repositories.Results = append(repositories.Results, r.Results...)
			repositories.Next = r.Next
		} else {
			break
		}
	}
	return repositories, nil
}

// getCommits TODO
func getCommits(url, reponame, token string) (Commits, error) {
	body, err := doRequest(fmt.Sprintf("%s%s/log", url, reponame), token)
	if err != nil {
		return Commits{}, err
	}

	commits := Commits{}
	err = json.Unmarshal(body, &commits)

	return commits, nil
}

// getRefs TODO
func getRefs(url, name, token string) (Refs, error) {
	body, err := doRequest(fmt.Sprintf("%s/%s/refs", url, name), token)
	if err != nil {
		return Refs{}, err
	}

	refs := Refs{}
	err = json.Unmarshal(body, &refs)

	for {
		if refs.Next != "" {
			body, err := doRequest(fmt.Sprintf("%s%s/refs/id=%s", url, name, refs.Next), token)
			if err != nil {
				return Refs{}, err
			}

			r := Refs{}

			err = json.Unmarshal(body, &r)
			if err != nil {
				return Refs{}, err
			}

			refs.Results = append(refs.Results, r.Results...)
			refs.Next = r.Next
		} else {
			break
		}
	}

	return refs, nil
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Sourcehut {
		err := repo.Filter.ParseDuration()
		if err != nil {
			log.Error().
				Str("stage", "bitbucket").
				Str("url", repo.URL).
				Msg(err.Error())
		}
		ran = true
		if repo.URL == "" {
			repo.URL = "https://git.sr.ht"
		}

		if !strings.HasSuffix(repo.URL, "/") {
			repo.URL += "/"
		}

		apiURL := fmt.Sprintf("%sapi/", repo.URL)

		token := repo.GetToken()

		if repo.User == "" {
			user := User{}
			body, err := doRequest(fmt.Sprintf("%suser", apiURL), token)
			if err != nil {
				log.Error().
					Str("stage", "sourcehut").
					Str("url", repo.URL).
					Msg("no user associated with this token")
				continue
			}

			err = json.Unmarshal(body, &user)
			if err != nil {
				log.Error().
					Str("stage", "sourcehut").
					Str("url", repo.URL).
					Msg("cannot unmarshal user")
				continue
			}
			repo.User = user.Name
		}

		log.Info().
			Str("stage", "sourcehut").
			Str("url", repo.URL).
			Msgf("grabbing repositories from %s", repo.User)

		if repo.User != "" {
			if !strings.HasPrefix(repo.User, "~") {
				repo.User = fmt.Sprintf("~%s", repo.User)
			}
		}

		apiURL = fmt.Sprintf("%sapi/%s/repos/", repo.URL, repo.User)

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)

		repositories, err := getRepos(apiURL, token)
		if err != nil {
			log.Error().
				Str("stage", "sourcehut").
				Str("url", repo.URL).
				Msg(err.Error())
		}

		for _, r := range repositories.Results {
			repoURL := fmt.Sprintf("%s%s/%s", repo.URL, repo.User, r.Name)
			sshURL := fmt.Sprintf("git@%s:%s/%s", types.GetHost(repo.URL), r.Owner.CanonicalName, r.Name)

			refs, err := getRefs(apiURL, r.Name, token)
			if err != nil {
				log.Error().
					Str("stage", "sourcehut").
					Str("url", repo.URL).
					Msg(err.Error())
			}

			head := ""
			for _, ref := range refs.Results {
				if strings.HasPrefix("refs/heads/", ref.Name) {
					head = strings.TrimLeft(ref.Name, "refs/heads/")
					break
				}
			}

			commits, err := getCommits(apiURL, r.Name, token)
			if err != nil {
				log.Error().
					Str("stage", "sourcehut").
					Str("url", repo.URL).
					Msg(err.Error())
			} else {
				if len(commits.Results) > 0 {
					if time.Since(commits.Results[0].Timestamp) > repo.Filter.LastActivityDuration && repo.Filter.LastActivityDuration != 0 {
						continue
					}
				}
			}

			if include[r.Name] {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           repoURL,
					SSHURL:        sshURL,
					Token:         token,
					Defaultbranch: refs.Results[0].Name,
					Origin:        repo,
					Owner:         r.Owner.Name,
					Hoster:        types.GetHost(repo.URL),
				})
				if repo.Wiki {
					repos = append(repos, types.Repo{
						Name:          r.Name + ".-docs",
						URL:           repoURL + "-docs",
						SSHURL:        sshURL + "-docs",
						Token:         token,
						Defaultbranch: head,
						Origin:        repo,
						Owner:         r.Owner.Name,
						Hoster:        types.GetHost(repo.URL),
					})
				}

				continue
			}

			if exclude[r.Name] {
				continue
			}

			if len(include) == 0 {
				repos = append(repos, types.Repo{
					Name:          r.Name,
					URL:           repoURL,
					SSHURL:        sshURL,
					Token:         token,
					Defaultbranch: head,
					Origin:        repo,
					Owner:         r.Owner.Name,
					Hoster:        types.GetHost(repo.URL),
				})
				if repo.Wiki {
					refs, err := getRefs(apiURL, fmt.Sprintf("%s-docs", r.Name), token)
					if err != nil {
						continue
					}
					if len(refs.Results) > 0 {
						head = ""
						for _, ref := range refs.Results {
							if strings.HasPrefix("refs/heads/", ref.Name) {
								head = strings.TrimLeft(ref.Name, "refs/heads/")
								break
							}
						}

						repos = append(repos, types.Repo{
							Name:          r.Name + "-docs",
							URL:           repoURL + "-docs",
							SSHURL:        sshURL + "-docs",
							Token:         token,
							Defaultbranch: head,
							Origin:        repo,
							Owner:         r.Owner.Name,
							Hoster:        types.GetHost(repo.URL),
						})
					}
				}
			}
		}
	}

	return repos, ran
}

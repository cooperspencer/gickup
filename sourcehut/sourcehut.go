package sourcehut

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
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

// postRequest TODO
func postRequest(url string, postbody []byte, token string) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(postbody))

	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Add("Content-Type", "application/json")

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

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Sourcehut {
		if repo.URL == "" {
			repo.URL = "https://git.sr.ht"
		}

		if !strings.HasSuffix(repo.URL, "/") {
			repo.URL += "/"
		}

		sub = logger.CreateSubLogger("stage", "sourcehut", "url", repo.URL)
		err := repo.Filter.ParseDuration()
		if err != nil {
			sub.Warn().
				Msg(err.Error())
		}
		ran = true

		apiURL := fmt.Sprintf("%sapi/", repo.URL)

		token := repo.GetToken()

		if repo.User == "" {
			user := User{}
			body, err := doRequest(fmt.Sprintf("%suser", apiURL), token)
			if err != nil {
				sub.Error().
					Msg("no user associated with this token")
				continue
			}

			err = json.Unmarshal(body, &user)
			if err != nil {
				sub.Error().
					Msg("cannot unmarshal user")
				continue
			}
			repo.User = user.Name
		}

		sub.Info().
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
			sub.Error().
				Msg(err.Error())
		}

		if len(repositories.Results) == 0 {
			sub.Error().Msgf("couldn't find any repositories for user %s", repo.User)
			break
		}

		for _, r := range repositories.Results {
			repoURL := fmt.Sprintf("%s%s/%s", repo.URL, repo.User, r.Name)
			sshURL := fmt.Sprintf("git@%s:%s/%s", types.GetHost(repo.URL), r.Owner.CanonicalName, r.Name)
			sub.Debug().Msg(repoURL)

			commits, err := getCommits(apiURL, r.Name, token)
			if err != nil {
				sub.Error().
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
					Name:        r.Name,
					URL:         repoURL,
					SSHURL:      sshURL,
					Token:       token,
					Origin:      repo,
					Owner:       r.Owner.Name,
					Hoster:      types.GetHost(repo.URL),
					Description: r.Description,
					Private:     r.Visibility == "private",
				})
				if repo.Wiki {
					repos = append(repos, types.Repo{
						Name:        r.Name + "-docs",
						URL:         repoURL + "-docs",
						SSHURL:      sshURL + "-docs",
						Token:       token,
						Origin:      repo,
						Owner:       r.Owner.Name,
						Hoster:      types.GetHost(repo.URL),
						Description: r.Description,
						Private:     r.Visibility == "private",
					})
				}

				continue
			}

			if exclude[r.Name] {
				continue
			}

			if len(include) == 0 {
				repos = append(repos, types.Repo{
					Name:        r.Name,
					URL:         repoURL,
					SSHURL:      sshURL,
					Token:       token,
					Origin:      repo,
					Owner:       r.Owner.Name,
					Hoster:      types.GetHost(repo.URL),
					Description: r.Description,
					Private:     r.Visibility == "private",
				})
				if repo.Wiki {
					repos = append(repos, types.Repo{
						Name:        r.Name + "-docs",
						URL:         repoURL + "-docs",
						SSHURL:      sshURL + "-docs",
						Token:       token,
						Origin:      repo,
						Owner:       r.Owner.Name,
						Hoster:      types.GetHost(repo.URL),
						Description: r.Description,
						Private:     r.Visibility == "private",
					})
				}
			}
		}
	}

	return repos, ran
}

func GetOrCreate(destination types.GenRepo, repo types.Repo) (string, error) {
	if destination.URL == "" {
		destination.URL = "https://git.sr.ht"
	}

	sub = logger.CreateSubLogger("stage", "sourcehut", "url", destination.URL)

	if !strings.HasSuffix(destination.URL, "/") {
		destination.URL += "/"
	}

	repository := Repository{}
	body, err := doRequest(fmt.Sprintf("%sapi/repos/%s", destination.URL, repo.Name), destination.GetToken())
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &repository)
	if err != nil {
		return "", err
	}
	if repository.Name == "" {
		if destination.Visibility.Repositories != "public" && destination.Visibility.Repositories != "private" && destination.Visibility.Repositories != "unlisted" {
			destination.Visibility.Repositories = "public"
		}
		postRepo := PostRepo{Name: repo.Name, Visibility: destination.Visibility.Repositories}
		postBody, err := json.Marshal(postRepo)
		if err != nil {
			return "", err
		}
		body, err := postRequest(fmt.Sprintf("%sapi/repos", destination.URL), postBody, destination.GetToken())
		if err != nil {
			return "", err
		}
		err = json.Unmarshal(body, &repository)
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("git@%s:%s/%s", types.GetHost(destination.URL), repository.Owner.CanonicalName, repo.Name), nil
}

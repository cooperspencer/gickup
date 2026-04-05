package sourcehut

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	graphqlclient "github.com/hasura/go-graphql-client"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
)

const (
	defaultSourcehutURL = "https://git.sr.ht"
)

func normalizeBearerToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return token
	}

	lower := strings.ToLower(token)
	if strings.HasPrefix(lower, "bearer ") {
		return strings.TrimSpace(token[len("bearer "):])
	}

	if strings.HasPrefix(lower, "token ") {
		return strings.TrimSpace(token[len("token "):])
	}

	return token
}

func normalizeURL(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return defaultSourcehutURL
	}

	return strings.TrimRight(strings.TrimSpace(rawURL), "/")
}

func graphQLEndpoint(rawURL string) string {
	return fmt.Sprintf("%s/query", normalizeURL(rawURL))
}

func newGraphQLClient(endpoint, token string) *graphqlclient.Client {
	token = normalizeBearerToken(token)
	client := graphqlclient.NewClient(endpoint, &http.Client{})
	if token != "" {
		client = client.WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		})
	}
	return client
}

func execGraphQL(endpoint, token, query string, variables map[string]interface{}, dataTarget interface{}) error {
	client := newGraphQLClient(endpoint, token)
	raw, err := client.ExecRaw(context.Background(), query, variables)
	if err != nil {
		return err
	}
	if dataTarget == nil {
		return nil
	}
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return errors.New("sourcehut graphql returned no data")
	}
	return json.Unmarshal(raw, dataTarget)
}

func resolveSourcehutUsername(endpoint, token, configuredUser string) (string, error) {
	if configuredUser != "" {
		return strings.TrimPrefix(configuredUser, "~"), nil
	}

	query := `query { me { username } }`
	response := queryMe{}
	if err := execGraphQL(endpoint, token, query, nil, &response); err != nil {
		return "", err
	}

	if response.Me.Username == "" {
		return "", errors.New("no user associated with this token")
	}

	return response.Me.Username, nil
}

func getRepositoriesForUser(endpoint, token, username string) ([]repository, error) {
	query := `query($username: String!, $cursor: Cursor) {
		user(username: $username) {
			repositories(cursor: $cursor) {
				results {
					id
					created
					updated
					name
					description
					visibility
					owner {
						canonicalName
					}
				}
				cursor
			}
		}
	}`

	allRepos := []repository{}
	var cursor *string

	for {
		variables := map[string]interface{}{
			"username": strings.TrimPrefix(username, "~"),
			"cursor":   cursor,
		}

		response := queryUser{}
		if err := execGraphQL(endpoint, token, query, variables, &response); err != nil {
			return nil, err
		}

		if response.User == nil {
			return nil, fmt.Errorf("couldn't find sourcehut user %s", username)
		}

		allRepos = append(allRepos, response.User.Repositories.Results...)
		if response.User.Repositories.Cursor == nil {
			break
		}

		cursor = response.User.Repositories.Cursor
	}

	return allRepos, nil
}

func getRepositoryByName(endpoint, token, username, repoName string) (*repository, error) {
	if username != "" {
		query := `query($username: String!, $name: String!) {
			user(username: $username) {
				repository(name: $name) {
					id
					name
					owner {
						canonicalName
					}
				}
			}
		}`

		response := queryUser{}
		variables := map[string]interface{}{
			"username": strings.TrimPrefix(username, "~"),
			"name":     repoName,
		}

		if err := execGraphQL(endpoint, token, query, variables, &response); err != nil {
			return nil, err
		}

		if response.User == nil {
			return nil, fmt.Errorf("couldn't find sourcehut user %s", username)
		}

		return response.User.Repository, nil
	}

	query := `query($name: String!) {
		me {
			repository(name: $name) {
				id
				name
				owner {
					canonicalName
				}
			}
		}
	}`

	response := queryMe{}
	variables := map[string]interface{}{"name": repoName}
	if err := execGraphQL(endpoint, token, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Me.Repository, nil
}

func createRepository(endpoint, token string, repo types.Repo, visibility string) (*repository, error) {
	query := `mutation($name: String!, $visibility: Visibility!, $description: String) {
		createRepository(name: $name, visibility: $visibility, description: $description) {
			id
			name
			owner {
				canonicalName
			}
		}
	}`

	response := mutationCreateRepository{}
	variables := map[string]interface{}{
		"name":        repo.Name,
		"visibility":  visibility,
		"description": repo.Description,
	}

	if err := execGraphQL(endpoint, token, query, variables, &response); err != nil {
		return nil, err
	}

	if response.CreateRepository == nil {
		return nil, errors.New("sourcehut did not return a created repository")
	}

	return response.CreateRepository, nil
}

func mapVisibilityToGraphQLEnum(visibility string) string {
	switch strings.ToLower(strings.TrimSpace(visibility)) {
	case "private":
		return "PRIVATE"
	case "unlisted":
		return "UNLISTED"
	default:
		return "PUBLIC"
	}
}

func buildHTTPURL(baseURL, canonicalOwner, repoName string) string {
	return fmt.Sprintf("%s/%s/%s", normalizeURL(baseURL), canonicalOwner, repoName)
}

func buildSSHURL(baseURL, canonicalOwner, repoName string) string {
	return fmt.Sprintf("git@%s:%s/%s", types.GetHost(normalizeURL(baseURL)), canonicalOwner, repoName)
}

// Get TODO.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}
	for _, repo := range conf.Source.Sourcehut {
		repo.URL = normalizeURL(repo.URL)

		sub = logger.CreateSubLogger("stage", "sourcehut", "url", repo.URL)
		err := repo.Filter.ParseDuration()
		if err != nil {
			sub.Warn().
				Msg(err.Error())
		}
		ran = true

		endpoint := graphQLEndpoint(repo.URL)

		token := repo.GetToken()

		repo.User, err = resolveSourcehutUsername(endpoint, token, repo.User)
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		sub.Info().
			Msgf("grabbing repositories from %s", repo.User)

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)

		repositories, err := getRepositoriesForUser(endpoint, token, repo.User)
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		if len(repositories) == 0 {
			sub.Error().Msgf("couldn't find any repositories for user %s", repo.User)
			continue
		}

		for _, r := range repositories {
			ownerCanonicalName := r.Owner.CanonicalName
			if ownerCanonicalName == "" {
				ownerCanonicalName = fmt.Sprintf("~%s", strings.TrimPrefix(repo.User, "~"))
			}

			repoURL := buildHTTPURL(repo.URL, ownerCanonicalName, r.Name)
			sshURL := buildSSHURL(repo.URL, ownerCanonicalName, r.Name)
			sub.Debug().Msg(repoURL)

			if repo.Filter.LastActivityDuration != 0 {
				if !r.Updated.IsZero() && time.Since(r.Updated) > repo.Filter.LastActivityDuration {
					continue
				}
			}

			ownerName := strings.TrimPrefix(ownerCanonicalName, "~")
			isPrivate := strings.EqualFold(r.Visibility, "PRIVATE") || strings.EqualFold(r.Visibility, "private")

			if include[r.Name] {
				repos = append(repos, types.Repo{
					Name:        r.Name,
					URL:         repoURL,
					SSHURL:      sshURL,
					Token:       token,
					Origin:      repo,
					Owner:       ownerName,
					Hoster:      types.GetHost(repo.URL),
					Description: r.Description,
					Private:     isPrivate,
				})
				if repo.Wiki {
					repos = append(repos, types.Repo{
						Name:        r.Name + "-docs",
						URL:         repoURL + "-docs",
						SSHURL:      sshURL + "-docs",
						Token:       token,
						Origin:      repo,
						Owner:       ownerName,
						Hoster:      types.GetHost(repo.URL),
						Description: r.Description,
						Private:     isPrivate,
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
					Owner:       ownerName,
					Hoster:      types.GetHost(repo.URL),
					Description: r.Description,
					Private:     isPrivate,
				})
				if repo.Wiki {
					repos = append(repos, types.Repo{
						Name:        r.Name + "-docs",
						URL:         repoURL + "-docs",
						SSHURL:      sshURL + "-docs",
						Token:       token,
						Origin:      repo,
						Owner:       ownerName,
						Hoster:      types.GetHost(repo.URL),
						Description: r.Description,
						Private:     isPrivate,
					})
				}
			}
		}
	}

	return repos, ran
}

func GetOrCreate(destination types.GenRepo, repo types.Repo) (string, error) {
	destination.URL = normalizeURL(destination.URL)

	sub = logger.CreateSubLogger("stage", "sourcehut", "url", destination.URL)

	token := destination.GetToken()
	endpoint := graphQLEndpoint(destination.URL)
	configuredUser := strings.TrimPrefix(destination.User, "~")

	remoteRepo, err := getRepositoryByName(endpoint, token, configuredUser, repo.Name)
	if err != nil {
		return "", err
	}

	if remoteRepo == nil {
		remoteRepo, err = createRepository(endpoint, token, repo, mapVisibilityToGraphQLEnum(destination.Visibility.Repositories))
		if err != nil {
			return "", err
		}
	}

	if remoteRepo == nil || remoteRepo.Owner.CanonicalName == "" {
		return "", errors.New("sourcehut repository owner could not be determined")
	}

	return buildSSHURL(destination.URL, remoteRepo.Owner.CanonicalName, repo.Name), nil
}

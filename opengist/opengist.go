package opengist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var sub zerolog.Logger

// perPage is the page size used when paginating the Opengist API.
const perPage = 50

// gist mirrors the relevant fields of Opengist's GistSimple API object
// (internal/web/handlers/api/v1/types/gist.go upstream).
type gist struct {
	ID          string `json:"id"`
	SlugURL     string `json:"slug_url"`
	Owner       user   `json:"owner"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Visibility  string `json:"visibility"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
}

// user mirrors the fields of Opengist's SimpleUser/PrivateUser.
type user struct {
	Username string `json:"username"`
}

// client is a minimal Opengist REST client for the API (Bearer token auth) over net/http.
type client struct {
	base  string
	token string
	http  *http.Client
}

func (c *client) get(path string, query url.Values) ([]byte, error) {
	endpoint := c.base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

// getUser returns the token owner (GET /api/user).
func (c *client) getUser() (user, error) {
	var u user
	body, err := c.get("/api/user", nil)
	if err != nil {
		return u, err
	}
	return u, json.Unmarshal(body, &u)
}

// listGists pages through a gist-listing endpoint and returns everything.
func (c *client) listGists(path string) ([]gist, error) {
	all := []gist{}
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(perPage))

		body, err := c.get(path, query)
		if err != nil {
			return nil, err
		}

		batch := []gist{}
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}

		all = append(all, batch...)

		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

// Get lists gists from every configured Opengist source and
// turns them into repositories gickup can clone and back up.
//
// Listing endpoint is chosen per source entry:
//   - user set              -> GET /api/users/{user}/gists (user's gists)
//   - user empty + token    -> GET /api/gists              (token owner's own gists)
//   - user empty + no token -> GET /api/gists/public       (public gists only)
//
// Opengist's git-over-HTTP basic auth requires the username to match the token owner,
// so private gists are cloned as that user regardless of which endpoint listed them.
// Public/unlisted gists clone anonymously when no token is set.
func Get(conf *types.Conf) ([]types.Repo, bool) {
	ran := false
	repos := []types.Repo{}

	if len(conf.Source.Opengist) > 0 {
		forceGitUserAgent()
	}

	for _, repo := range conf.Source.Opengist {
		ran = true

		if repo.URL == "" {
			log.Error().
				Str("stage", "opengist").
				Msg("no url configured")
			continue
		}

		base := strings.TrimRight(repo.URL, "/")
		sub = logger.CreateSubLogger("stage", "opengist", "url", base)

		token := repo.GetToken()
		cl := &client{base: base, token: token, http: &http.Client{Timeout: 30 * time.Second}}

		// The clone username must match the token owner for private gists.
		tokenOwner := ""
		if token != "" {
			u, err := cl.getUser()
			if err != nil {
				sub.Warn().
					Msgf("couldn't determine token owner, private gists may not clone: %s", err.Error())
			} else {
				tokenOwner = u.Username
			}
		}

		var listPath string
		switch {
		case repo.User != "":
			listPath = "/api/users/" + url.PathEscape(repo.User) + "/gists"
			sub.Info().
				Msgf("grabbing gists from %s", repo.User)
		case token != "":
			listPath = "/api/gists"
			sub.Info().
				Msg("grabbing my gists")
		default:
			listPath = "/api/gists/public"
			sub.Info().
				Msg("grabbing public gists")
		}

		gists, err := cl.listGists(listPath)
		if err != nil {
			sub.Error().
				Msg(err.Error())
			continue
		}

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)

		for _, g := range gists {
			name := g.SlugURL
			if name == "" {
				name = g.ID
			}

			if exclude[name] {
				continue
			}
			if len(repo.Include) > 0 && !include[name] {
				continue
			}

			// Origin carries the clone credentials. Opengist wants the token
			// owner as the basic-auth username, so override it here.
			origin := repo
			if tokenOwner != "" {
				origin.User = tokenOwner
			}

			repos = append(repos, types.Repo{
				Name:        name,
				URL:         g.CloneURL,
				SSHURL:      g.SSHURL,
				Token:       token,
				Origin:      origin,
				Owner:       g.Owner.Username,
				Hoster:      types.GetHost(base),
				Description: g.Description,
				Private:     !g.Public,
			})
		}
	}

	return repos, ran
}

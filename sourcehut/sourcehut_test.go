package sourcehut

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cooperspencer/gickup/types"
)

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func newGraphQLServer(t *testing.T, handler func(*http.Request, graphQLRequest) map[string]interface{}) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" {
			t.Fatalf("path = %q, want /query", r.URL.Path)
		}

		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		resp := map[string]interface{}{"data": handler(r, req)}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
}

func TestNormalizeHelpers(t *testing.T) {
	t.Parallel()

	if got := normalizeBearerToken("  Bearer secret  "); got != "secret" {
		t.Fatalf("normalizeBearerToken() = %q, want secret", got)
	}

	if got := normalizeBearerToken("token secret"); got != "secret" {
		t.Fatalf("normalizeBearerToken() = %q, want secret", got)
	}

	if got := normalizeURL(" https://git.sr.ht/ "); got != "https://git.sr.ht" {
		t.Fatalf("normalizeURL() = %q, want trimmed URL", got)
	}

	if got := normalizeURL(""); got != defaultSourcehutURL {
		t.Fatalf("normalizeURL() = %q, want default URL", got)
	}

	if got := graphQLEndpoint("https://git.sr.ht/"); got != "https://git.sr.ht/query" {
		t.Fatalf("graphQLEndpoint() = %q", got)
	}

	if got := mapVisibilityToGraphQLEnum(" unlisted "); got != "UNLISTED" {
		t.Fatalf("mapVisibilityToGraphQLEnum() = %q, want UNLISTED", got)
	}

	if got := buildHTTPURL("https://git.sr.ht/", "~alice", "repo"); got != "https://git.sr.ht/~alice/repo" {
		t.Fatalf("buildHTTPURL() = %q", got)
	}

	if got := buildSSHURL("https://git.sr.ht/", "~alice", "repo"); got != "git@git.sr.ht:~alice/repo" {
		t.Fatalf("buildSSHURL() = %q", got)
	}
}

func TestResolveSourcehutUsernameUsesConfiguredUser(t *testing.T) {
	t.Parallel()

	got, err := resolveSourcehutUsername("https://unused.invalid/query", "ignored", "~alice")
	if err != nil {
		t.Fatalf("resolveSourcehutUsername() error = %v", err)
	}

	if got != "alice" {
		t.Fatalf("resolveSourcehutUsername() = %q, want alice", got)
	}
}

func TestResolveSourcehutUsernameQueriesGraphQL(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, func(r *http.Request, req graphQLRequest) map[string]interface{} {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("authorization = %q, want Bearer secret-token", got)
		}

		if !strings.Contains(req.Query, "me") {
			t.Fatalf("unexpected query: %s", req.Query)
		}

		return map[string]interface{}{"me": map[string]interface{}{"username": "carol"}}
	})
	defer server.Close()

	got, err := resolveSourcehutUsername(server.URL+"/query", "Token secret-token", "")
	if err != nil {
		t.Fatalf("resolveSourcehutUsername() error = %v", err)
	}

	if got != "carol" {
		t.Fatalf("resolveSourcehutUsername() = %q, want carol", got)
	}
}

func TestGetRepositoriesForUserPaginates(t *testing.T) {
	t.Parallel()

	updated := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339)
	server := newGraphQLServer(t, func(_ *http.Request, req graphQLRequest) map[string]interface{} {
		cursor, _ := req.Variables["cursor"].(string)
		if cursor == "" {
			next := "next-page"
			return map[string]interface{}{
				"user": map[string]interface{}{
					"repositories": map[string]interface{}{
						"results": []map[string]interface{}{{
							"id":          1,
							"name":        "repo-one",
							"description": "first",
							"visibility":  "PUBLIC",
							"created":     updated,
							"updated":     updated,
							"owner":       map[string]interface{}{"canonicalName": "~alice"},
						}},
						"cursor": next,
					},
				},
			}
		}

		if cursor != "next-page" {
			t.Fatalf("unexpected cursor: %q", cursor)
		}

		return map[string]interface{}{
			"user": map[string]interface{}{
				"repositories": map[string]interface{}{
					"results": []map[string]interface{}{{
						"id":          2,
						"name":        "repo-two",
						"description": "second",
						"visibility":  "PRIVATE",
						"created":     updated,
						"updated":     updated,
						"owner":       map[string]interface{}{"canonicalName": "~alice"},
					}},
					"cursor": nil,
				},
			},
		}
	})
	defer server.Close()

	repos, err := getRepositoriesForUser(server.URL+"/query", "secret-token", "alice")
	if err != nil {
		t.Fatalf("getRepositoriesForUser() error = %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}

	if repos[0].Name != "repo-one" || repos[1].Name != "repo-two" {
		t.Fatalf("unexpected repos: %#v", repos)
	}
}

func TestCreateRepository(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, func(_ *http.Request, req graphQLRequest) map[string]interface{} {
		if !strings.Contains(req.Query, "mutation") {
			t.Fatalf("unexpected query: %s", req.Query)
		}

		if got := fmt.Sprint(req.Variables["visibility"]); got != "PRIVATE" {
			t.Fatalf("visibility = %q, want PRIVATE", got)
		}

		return map[string]interface{}{
			"createRepository": map[string]interface{}{
				"id":    99,
				"name":  req.Variables["name"],
				"owner": map[string]interface{}{"canonicalName": "~alice"},
			},
		}
	})
	defer server.Close()

	repo, err := createRepository(server.URL+"/query", "secret-token", types.Repo{Name: "repo-one", Description: "mirror"}, "PRIVATE")
	if err != nil {
		t.Fatalf("createRepository() error = %v", err)
	}

	if repo == nil || repo.Name != "repo-one" || repo.Owner.CanonicalName != "~alice" {
		t.Fatalf("unexpected repository: %#v", repo)
	}
}

func TestGetBuildsRepositoriesAndWikiMirrors(t *testing.T) {
	t.Parallel()

	updated := time.Now().UTC().Format(time.RFC3339)
	server := newGraphQLServer(t, func(_ *http.Request, req graphQLRequest) map[string]interface{} {
		if !strings.Contains(req.Query, "repositories") {
			t.Fatalf("unexpected query: %s", req.Query)
		}

		return map[string]interface{}{
			"user": map[string]interface{}{
				"repositories": map[string]interface{}{
					"results": []map[string]interface{}{{
						"id":          1,
						"name":        "repo-one",
						"description": "first repo",
						"visibility":  "PRIVATE",
						"created":     updated,
						"updated":     updated,
						"owner":       map[string]interface{}{"canonicalName": "~alice"},
					}},
					"cursor": nil,
				},
			},
		}
	})
	defer server.Close()

	conf := &types.Conf{
		Source: types.Source{
			Sourcehut: []types.GenRepo{{
				URL:     server.URL,
				User:    "alice",
				Include: []string{"repo-one"},
				Wiki:    true,
			}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected sourcehut adapter to run")
	}

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos including wiki, got %d", len(repos))
	}

	if repos[0].Name != "repo-one" || !repos[0].Private || repos[0].Description != "first repo" {
		t.Fatalf("unexpected primary repo: %#v", repos[0])
	}

	if repos[1].Name != "repo-one-docs" || !strings.HasSuffix(repos[1].URL, "/repo-one-docs") {
		t.Fatalf("unexpected wiki repo: %#v", repos[1])
	}
}

func TestGetOrCreateReturnsExistingRepositoryURL(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, func(_ *http.Request, req graphQLRequest) map[string]interface{} {
		if !strings.Contains(req.Query, "repository(name") {
			t.Fatalf("unexpected query: %s", req.Query)
		}

		return map[string]interface{}{
			"user": map[string]interface{}{
				"repository": map[string]interface{}{
					"id":    7,
					"name":  "repo-one",
					"owner": map[string]interface{}{"canonicalName": "~alice"},
				},
			},
		}
	})
	defer server.Close()

	url, err := GetOrCreate(types.GenRepo{URL: server.URL, User: "alice", Token: "secret-token"}, types.Repo{Name: "repo-one"})
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}

	if !strings.HasPrefix(url, "git@") || !strings.HasSuffix(url, ":~alice/repo-one") {
		t.Fatalf("unexpected ssh url: %q", url)
	}
}

func TestGetOrCreateCreatesMissingRepository(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, func(_ *http.Request, req graphQLRequest) map[string]interface{} {
		switch {
		case strings.Contains(req.Query, "repository(name"):
			return map[string]interface{}{
				"user": map[string]interface{}{
					"repository": nil,
				},
			}
		case strings.Contains(req.Query, "mutation"):
			return map[string]interface{}{
				"createRepository": map[string]interface{}{
					"id":    8,
					"name":  req.Variables["name"],
					"owner": map[string]interface{}{"canonicalName": "~alice"},
				},
			}
		default:
			t.Fatalf("unexpected query: %s", req.Query)
			return nil
		}
	})
	defer server.Close()

	url, err := GetOrCreate(types.GenRepo{
		URL:        server.URL,
		User:       "alice",
		Token:      "secret-token",
		Visibility: types.Visibility{Repositories: "private"},
	}, types.Repo{Name: "repo-two", Description: "created during test"})
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}

	if !strings.HasPrefix(url, "git@") || !strings.HasSuffix(url, ":~alice/repo-two") {
		t.Fatalf("unexpected ssh url: %q", url)
	}
}

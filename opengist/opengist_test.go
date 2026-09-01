package opengist

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

// gistPayload builds an Opengist-shaped gist JSON object.
func gistPayload(slug, owner string, public bool) map[string]interface{} {
	return map[string]interface{}{
		"id":          slug,
		"slug_url":    slug,
		"owner":       map[string]interface{}{"username": owner},
		"title":       slug,
		"description": "desc-" + slug,
		"public":      public,
		"visibility":  map[bool]string{true: "public", false: "private"}[public],
		"clone_url":   fmt.Sprintf("http://opengist.example/%s/%s.git", owner, slug),
		"ssh_url":     fmt.Sprintf("git@opengist.example:%s/%s.git", owner, slug),
	}
}

// newServer returns an httptest server that serves a single page of gists on
// the given path plus a token-owner endpoint, then an empty second page.
func newServer(t *testing.T, path string, gists []map[string]interface{}) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/user", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"username": "tokenowner"})
	})

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page > 1 {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(gists)
	})

	return httptest.NewServer(mux)
}

func TestGetOwnGists(t *testing.T) {
	srv := newServer(t, "/api/gists", []map[string]interface{}{
		gistPayload("alpha", "alice", true),
		gistPayload("beta", "alice", false),
	})
	defer srv.Close()

	conf := &types.Conf{
		Source: types.Source{
			Opengist: []types.GenRepo{{URL: srv.URL, Token: "og_token"}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}

	r := repos[0]
	if r.Name != "alpha" || r.Owner != "alice" {
		t.Fatalf("unexpected repo: %#v", r)
	}
	if r.URL != "http://opengist.example/alice/alpha.git" {
		t.Fatalf("unexpected clone url: %q", r.URL)
	}
	// Clone auth must use the token owner as username (Opengist requirement).
	if r.Origin.User != "tokenowner" {
		t.Fatalf("expected clone user tokenowner, got %q", r.Origin.User)
	}
	if repos[1].Private != true {
		t.Fatalf("expected beta to be private")
	}
}

func TestGetPublicGistsAnonymous(t *testing.T) {
	srv := newServer(t, "/api/gists/public", []map[string]interface{}{
		gistPayload("pub", "bob", true),
	})
	defer srv.Close()

	conf := &types.Conf{
		Source: types.Source{
			Opengist: []types.GenRepo{{URL: srv.URL}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Token != "" || repos[0].Origin.User != "" {
		t.Fatalf("expected anonymous clone, got token=%q user=%q", repos[0].Token, repos[0].Origin.User)
	}
}

func TestGetUserGists(t *testing.T) {
	srv := newServer(t, "/api/users/carol/gists", []map[string]interface{}{
		gistPayload("one", "carol", true),
	})
	defer srv.Close()

	conf := &types.Conf{
		Source: types.Source{
			Opengist: []types.GenRepo{{URL: srv.URL, Token: "og_token", User: "carol"}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}
	if len(repos) != 1 || repos[0].Owner != "carol" {
		t.Fatalf("unexpected repos: %#v", repos)
	}
}

func TestGetIncludeExclude(t *testing.T) {
	gists := []map[string]interface{}{
		gistPayload("keep", "alice", true),
		gistPayload("drop", "alice", true),
	}

	t.Run("include", func(t *testing.T) {
		srv := newServer(t, "/api/gists", gists)
		defer srv.Close()
		conf := &types.Conf{Source: types.Source{Opengist: []types.GenRepo{{
			URL: srv.URL, Token: "og_token", Include: []string{"keep"},
		}}}}
		repos, _ := Get(conf)
		if len(repos) != 1 || repos[0].Name != "keep" {
			t.Fatalf("include filter failed: %#v", repos)
		}
	})

	t.Run("exclude", func(t *testing.T) {
		srv := newServer(t, "/api/gists", gists)
		defer srv.Close()
		conf := &types.Conf{Source: types.Source{Opengist: []types.GenRepo{{
			URL: srv.URL, Token: "og_token", Exclude: []string{"drop"},
		}}}}
		repos, _ := Get(conf)
		if len(repos) != 1 || repos[0].Name != "keep" {
			t.Fatalf("exclude filter failed: %#v", repos)
		}
	})
}

func TestGetSkipsMissingURL(t *testing.T) {
	conf := &types.Conf{Source: types.Source{Opengist: []types.GenRepo{{}}}}
	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}
	if len(repos) != 0 {
		t.Fatalf("expected no repositories, got %d", len(repos))
	}
}

func TestUserAgentTransportRewrites(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	rt := userAgentTransport{base: http.DefaultTransport}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "go-git/5.x")
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	resp.Body.Close()

	if got != gitUserAgent {
		t.Fatalf("expected User-Agent %q, got %q", gitUserAgent, got)
	}
}

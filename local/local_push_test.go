package local

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

//nolint:paralleltest // CreateRemotePush assigns the package-level logger.
func TestCreateRemotePushUsesProvidedToken(t *testing.T) {
	const want = "resolved-token-for-test"

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree() error = %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, err := wt.Commit("seed", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Now()},
	}); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	var passwordsMu sync.Mutex
	var passwords []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, password, _ := r.BasicAuth()
		passwordsMu.Lock()
		passwords = append(passwords, password)
		passwordsMu.Unlock()
		w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	for _, configuredToken := range []string{"", "unused-configured-token"} {
		destination := types.GenRepo{Token: configuredToken}
		err := CreateRemotePush(repo, destination, server.URL+"/org/repo.git", want, false)
		if !errors.Is(err, transport.ErrAuthenticationRequired) {
			t.Fatalf("push error = %v, want authentication required", err)
		}
		passwordsMu.Lock()
		gotPasswords := passwords
		passwords = nil
		passwordsMu.Unlock()
		if len(gotPasswords) == 0 {
			t.Fatal("push did not reach the server")
		}
		for _, got := range gotPasswords {
			if got != want {
				t.Fatalf("push password = %q, want %q", got, want)
			}
		}
	}

	reopened, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	remotes, err := reopened.Remotes()
	if err != nil {
		t.Fatal(err)
	}
	for _, remote := range remotes {
		urls := remote.Config().URLs
		if len(urls) != 1 || urls[0] != server.URL+"/org/repo.git" {
			t.Fatalf("persisted remote URLs = %v, want original credential-free URL", urls)
		}
	}
}

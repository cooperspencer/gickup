package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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
)

//nolint:paralleltest // backup calls providers that assign package-level loggers.
func TestBackupGitHubAppPushCredential(t *testing.T) {
	const wantToken = "installation-token-for-test"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "app.pem")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), 0o600); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	source, err := git.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := source.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Commit("seed", &git.CommitOptions{
		AllowEmptyCommits: true,
		Author:            &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	var passwordsMu sync.Mutex
	var passwords []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v3/app/installations/42/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"token": wantToken, "expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
			})
		case "GET /api/v3/orgs/example":
			_, _ = w.Write([]byte(`{"login":"example"}`))
		case "GET /api/v3/repos/example/repo":
			_ = json.NewEncoder(w).Encode(map[string]string{"clone_url": "http://" + r.Host + "/example/repo.git"})
		case "GET /example/repo.git/info/refs":
			_, password, _ := r.BasicAuth()
			passwordsMu.Lock()
			passwords = append(passwords, password)
			passwordsMu.Unlock()
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	backup([]types.Repo{{Name: "repo", URL: sourceDir}}, &types.Conf{
		Destination: types.Destination{Github: []types.GenRepo{{
			URL: server.URL, Organization: "example",
			AppID: 1, AppInstallationID: 42, AppPrivateKeyFile: keyFile,
		}}},
	})
	passwordsMu.Lock()
	gotPasswords := passwords
	passwords = nil
	passwordsMu.Unlock()
	if len(gotPasswords) == 0 {
		t.Fatal("backup did not attempt a push")
	}
	for _, got := range gotPasswords {
		if got != wantToken {
			t.Fatalf("backup push password = %q, want resolved installation token", got)
		}
	}
}

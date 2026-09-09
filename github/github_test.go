package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cooperspencer/gickup/types"
)

func TestNewGithubClientUnauthenticated(t *testing.T) {
	t.Parallel()

	repo := types.GenRepo{} // no token, no app auth

	client, token, err := newGithubClient(context.Background(), repo)
	if err != nil {
		t.Fatalf("newGithubClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestNewGithubClientWithToken(t *testing.T) {
	t.Setenv("GITHUB_TEST_TOKEN", "my-personal-token")

	repo := types.GenRepo{Token: "GITHUB_TEST_TOKEN"}

	client, token, err := newGithubClient(context.Background(), repo)
	if err != nil {
		t.Fatalf("newGithubClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if token != "my-personal-token" {
		t.Fatalf("token = %q, want %q", token, "my-personal-token")
	}
}

func TestNewGithubClientAppAuthInvalidKeyFile(t *testing.T) {
	t.Parallel()

	// A file that exists but contains an invalid PEM key.
	keyFile := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(keyFile, []byte("not-a-valid-pem-key"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	repo := types.GenRepo{
		AppID:             1,
		AppInstallationID: 2,
		AppPrivateKeyFile: keyFile,
	}

	_, _, err := newGithubClient(context.Background(), repo)
	if err == nil {
		t.Fatal("expected error for invalid App private key file")
	}
}

func TestNewGithubClientAppAuthMissingKeyFile(t *testing.T) {
	t.Parallel()

	repo := types.GenRepo{
		AppID:             1,
		AppInstallationID: 2,
		AppPrivateKeyFile: "/nonexistent/path/key.pem",
	}

	_, _, err := newGithubClient(context.Background(), repo)
	if err == nil {
		t.Fatal("expected error when App private key file does not exist")
	}
}

//nolint:paralleltest // Replaces http.DefaultTransport and calls code with a package-level logger.
func TestGetOrCreateReturnsResolvedToken(t *testing.T) {
	const wantToken = "installation-token-for-test"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	keypath := filepath.Join(t.TempDir(), "app.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(keypath, pemBytes, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte(wantToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_DESTINATION_TEST_TOKEN", wantToken)

	for _, instance := range []struct {
		name, url, apiURL, cloneURL string
	}{
		{"github.com", "", "https://api.github.com", "https://github.com/example/repo.git"},
		{"enterprise-server", "https://github.example.com/", "https://github.example.com/api/v3", "https://github.example.com/example/repo.git"},
	} {
		for _, tc := range []struct {
			name, auth, failAt string
			create             bool
		}{
			{name: "app-existing", auth: "app"},
			{name: "app-create", auth: "app", create: true},
			{name: "pat-existing", auth: "pat"},
			{name: "pat-create", auth: "pat", create: true},
			{name: "environment-token", auth: "environment"},
			{name: "token-file", auth: "file"},
			{name: "token-error", auth: "app", failAt: "/app/installations/4242/access_tokens"},
			{name: "organization-error", auth: "app", failAt: "/orgs/example"},
			{name: "repository-error", auth: "app", failAt: "/repos/example/repo"},
			{name: "creation-error", auth: "app", create: true, failAt: "/orgs/example/repos"},
		} {
			t.Run(instance.name+"/"+tc.name, func(t *testing.T) {
				destination := types.GenRepo{URL: instance.url, Organization: "example"}
				switch tc.auth {
				case "app":
					destination.AppID = 99
					destination.AppInstallationID = 4242
					destination.AppPrivateKeyFile = keypath
					destination.Token = "unused-configured-token"
				case "pat":
					destination.Token = wantToken
				case "environment":
					destination.Token = "GITHUB_DESTINATION_TEST_TOKEN"
				case "file":
					destination.TokenFile = tokenFile
				}

				created, failed := false, false
				// These fixtures rely on clients using http.DefaultTransport to intercept github.com requests.
				oldTransport := http.DefaultTransport
				t.Cleanup(func() { http.DefaultTransport = oldTransport })
				http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					w := httptest.NewRecorder()
					w.Header().Set("Content-Type", "application/json")
					if !strings.HasPrefix(r.URL.String(), instance.apiURL+"/") {
						t.Errorf("unexpected API URL: %s", r.URL)
						w.WriteHeader(http.StatusNotFound)
						return w.Result(), nil
					}
					endpoint := strings.TrimPrefix(r.URL.String(), instance.apiURL)
					if endpoint == "/app/installations/4242/access_tokens" {
						if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
							t.Error("installation token request must use POST and App JWT authentication")
						}
					} else if !strings.HasSuffix(r.Header.Get("Authorization"), " "+wantToken) {
						t.Error("API request did not use the resolved token")
					}
					if endpoint == tc.failAt {
						failed = true
						w.WriteHeader(http.StatusForbidden)
						return w.Result(), nil
					}
					switch r.Method + " " + endpoint {
					case "POST /app/installations/4242/access_tokens":
						w.WriteHeader(http.StatusCreated)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"token": wantToken, "expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
						})
					case "GET /orgs/example":
						_, _ = w.WriteString(`{"login":"example"}`)
					case "GET /repos/example/repo":
						if tc.create {
							w.WriteHeader(http.StatusNotFound)
							_, _ = w.WriteString(`{"message":"Not Found"}`)
						} else {
							_ = json.NewEncoder(w).Encode(map[string]string{"clone_url": instance.cloneURL})
						}
					case "POST /orgs/example/repos":
						created = true
						w.WriteHeader(http.StatusCreated)
						_ = json.NewEncoder(w).Encode(map[string]string{"clone_url": instance.cloneURL})
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						w.WriteHeader(http.StatusNotFound)
					}
					return w.Result(), nil
				})

				cloneURL, token, err := GetOrCreate(destination, types.Repo{Name: "repo"})
				if tc.failAt != "" {
					if !failed || err == nil || cloneURL != "" || token != "" {
						t.Fatalf("API error returned (%q, %q, %v), want empty URL and token with an error", cloneURL, token, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if cloneURL != instance.cloneURL || token != wantToken {
					t.Fatalf("GetOrCreate() = (%q, %q), want (%q, %q)", cloneURL, token, instance.cloneURL, wantToken)
				}
				if created != tc.create {
					t.Fatalf("repository created = %v, want %v", created, tc.create)
				}
			})
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

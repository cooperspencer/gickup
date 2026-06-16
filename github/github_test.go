package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

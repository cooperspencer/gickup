package local

import (
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestRandomStringLengthAndCharset(t *testing.T) {
	t.Parallel()

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	got := RandomString(64)
	if len(got) != 64 {
		t.Fatalf("len(RandomString()) = %d, want 64", len(got))
	}

	for _, r := range got {
		if !strings.ContainsRune(charset, r) {
			t.Fatalf("unexpected rune %q in %q", r, got)
		}
	}
}

func TestRandomStringZeroLength(t *testing.T) {
	t.Parallel()

	if got := RandomString(0); got != "" {
		t.Fatalf("RandomString(0) = %q, want empty string", got)
	}
}

func TestTokenAuth_NoToken(t *testing.T) {
	t.Parallel()

	repo := types.Repo{Token: ""}
	if got := tokenAuth(repo); got != nil {
		t.Fatalf("tokenAuth with empty token = %v, want nil", got)
	}
}

func TestTokenAuth_NoTokenUser(t *testing.T) {
	t.Parallel()

	repo := types.Repo{
		Token:       "mytoken",
		NoTokenUser: true,
	}
	got := tokenAuth(repo)
	if got == nil {
		t.Fatal("tokenAuth returned nil, want *BasicAuth")
	}
	if got.Username != "x-access-token" {
		t.Errorf("Username = %q, want %q", got.Username, "x-access-token")
	}
	if got.Password != "mytoken" {
		t.Errorf("Password = %q, want %q", got.Password, "mytoken")
	}
}

func TestTokenAuth_WithTokenUser(t *testing.T) {
	t.Parallel()

	repo := types.Repo{
		Token:       "mytoken",
		NoTokenUser: false,
		Origin: types.GenRepo{
			User: "octocat",
		},
	}
	got := tokenAuth(repo)
	if got == nil {
		t.Fatal("tokenAuth returned nil, want *BasicAuth")
	}
	if got.Username != "octocat" {
		t.Errorf("Username = %q, want %q", got.Username, "octocat")
	}
	if got.Password != "mytoken" {
		t.Errorf("Password = %q, want %q", got.Password, "mytoken")
	}
}

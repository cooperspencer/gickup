package local

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func runPrune(t *testing.T, backups []string, keep int) []string {
	t.Helper()

	parentdir := t.TempDir()
	for _, d := range backups {
		if err := os.Mkdir(filepath.Join(parentdir, d), 0o777); err != nil {
			t.Fatal(err)
		}
	}

	if err := pruneOldBackups(parentdir, "test/repo", types.Local{Keep: keep}); err != nil {
		t.Fatal(err)
	}

	files, err := os.ReadDir(parentdir)
	if err != nil {
		t.Fatal(err)
	}
	remaining := []string{}
	for _, f := range files {
		remaining = append(remaining, f.Name())
	}

	return remaining
}

func TestPruneKeepsNewestBackup(t *testing.T) {
	remaining := runPrune(t, []string{"1785061240", "1785115104"}, 1)

	want := []string{"1785115104"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsIssuesDirWithItsBackup(t *testing.T) {
	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785115111", "1785115111.issues"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsOtherDirectorys(t *testing.T) {
	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "dummy", "dummy.issue", "1785061241x.dummy", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785061241x.dummy", "1785115111", "1785115111.issues", "dummy", "dummy.issue"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneDeletesOtherTimestampedDirectorys(t *testing.T) {
	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "1785061241.dummy", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785115111", "1785115111.issues"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

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

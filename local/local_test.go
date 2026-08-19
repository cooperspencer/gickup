package local

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
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
	t.Parallel()

	remaining := runPrune(t, []string{"1785061240", "1785115104"}, 1)

	want := []string{"1785115104"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsIssuesDirWithItsBackup(t *testing.T) {
	t.Parallel()

	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785115111", "1785115111.issues"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsOtherDirectories(t *testing.T) {
	t.Parallel()

	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "dummy", "dummy.issue", "1785061241x.dummy", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785061241x.dummy", "1785115111", "1785115111.issues", "dummy", "dummy.issue"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneDeletesOtherTimestampedDirectories(t *testing.T) {
	t.Parallel()

	remaining := runPrune(t,
		[]string{"1785061241", "1785061241.issues", "1785061241.dummy", "1785115111", "1785115111.issues"}, 1)

	want := []string{"1785115111", "1785115111.issues"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsMultipleGenerations(t *testing.T) {
	t.Parallel()

	remaining := runPrune(t,
		[]string{"1785000000", "1785000000.issues", "1785061241", "1785061241.issues", "1785115111", "1785115111.issues"}, 2)

	want := []string{"1785061241", "1785061241.issues", "1785115111", "1785115111.issues"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
}

func TestPruneKeepsEverythingBelowKeepCount(t *testing.T) {
	t.Parallel()

	remaining := runPrune(t, []string{"1785115111", "1785115111.issues"}, 2)

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

func TestToGitCmdAuth_Nil(t *testing.T) {
	t.Parallel()

	if got := toGitCmdAuth(nil); got != nil {
		t.Fatalf("toGitCmdAuth(nil) = %v, want nil", got)
	}
}

func TestToGitCmdAuth_FromTokenAuth_GitHubApp(t *testing.T) {
	t.Parallel()

	repo := types.Repo{
		Token:       "ghs_testtoken123",
		NoTokenUser: true,
	}
	auth := tokenAuth(repo)
	gitAuth := toGitCmdAuth(auth)
	if gitAuth == nil {
		t.Fatal("toGitCmdAuth returned nil, want *gitcmd.Auth")
	}
	if gitAuth.Username != "x-access-token" {
		t.Errorf("Username = %q, want %q", gitAuth.Username, "x-access-token")
	}
	if gitAuth.Password != "ghs_testtoken123" {
		t.Errorf("Password = %q, want %q", gitAuth.Password, "ghs_testtoken123")
	}

	env := gitAuth.Env()
	if len(env) != 3 {
		t.Fatalf("expected 3 env vars, got %d: %v", len(env), env)
	}
	if env[0] != "GIT_CONFIG_COUNT=1" || env[1] != "GIT_CONFIG_KEY_0=http.extraHeader" {
		t.Errorf("unexpected env vars: %v", env)
	}
	if !strings.HasPrefix(env[2], "GIT_CONFIG_VALUE_0=Authorization: Basic ") {
		t.Errorf("unexpected GIT_CONFIG_VALUE_0: %s", env[2])
	}
}

func TestToGitCmdAuth_FromUsernamePassword(t *testing.T) {
	t.Parallel()

	auth := &http.BasicAuth{
		Username: "myuser",
		Password: "mypassword",
	}
	gitAuth := toGitCmdAuth(auth)
	if gitAuth == nil {
		t.Fatal("toGitCmdAuth returned nil, want *gitcmd.Auth")
	}
	if gitAuth.Username != "myuser" {
		t.Errorf("Username = %q, want %q", gitAuth.Username, "myuser")
	}
	if gitAuth.Password != "mypassword" {
		t.Errorf("Password = %q, want %q", gitAuth.Password, "mypassword")
	}
}

func TestToGitCmdAuth_SSHAuth(t *testing.T) {
	t.Parallel()

	auth := &ssh.PublicKeys{}
	if got := toGitCmdAuth(auth); got != nil {
		t.Fatalf("toGitCmdAuth(ssh) = %v, want nil", got)
	}
}

func TestSanitizeRemote_RewritesCredentialURL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r, err := git.PlainInit(dir, true)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://xyz:expired-token-123@github.com/owner/repo.git"},
	})
	if err != nil {
		t.Fatalf("failed to create remote: %v", err)
	}

	err = sanitizeRemote(r, "https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("sanitizeRemote returned error: %v", err)
	}

	// Reopen the repository from disk to verify persistence
	reopened, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("failed to reopen repo: %v", err)
	}

	cfg, err := reopened.Config()
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}

	rem, ok := cfg.Remotes["origin"]
	if !ok {
		t.Fatal("remote origin not found")
	}

	wantURLs := []string{"https://github.com/owner/repo.git"}
	if !reflect.DeepEqual(rem.URLs, wantURLs) {
		t.Errorf("remote URLs = %v, want %v", rem.URLs, wantURLs)
	}

	for _, u := range rem.URLs {
		if strings.Contains(u, "expired-token-123") || strings.Contains(u, "xyz:") || strings.Contains(u, "@") {
			t.Errorf("token or credentials leaked in remote URL: %s", u)
		}
	}
}

func TestSanitizeRemote_PreservesSSH(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r, err := git.PlainInit(dir, true)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:owner/repo.git"},
	})
	if err != nil {
		t.Fatalf("failed to create remote: %v", err)
	}

	err = sanitizeRemote(r, "https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("sanitizeRemote returned error: %v", err)
	}

	cfg, err := r.Config()
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}

	rem := cfg.Remotes["origin"]
	wantURLs := []string{"git@github.com:owner/repo.git"}
	if !reflect.DeepEqual(rem.URLs, wantURLs) {
		t.Errorf("SSH remote URLs = %v, want %v", rem.URLs, wantURLs)
	}
}

func TestSanitizeRemote_PreservesCleanURL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r, err := git.PlainInit(dir, true)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/owner/repo.git"},
	})
	if err != nil {
		t.Fatalf("failed to create remote: %v", err)
	}

	err = sanitizeRemote(r, "https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("sanitizeRemote returned error: %v", err)
	}

	cfg, err := r.Config()
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}

	rem := cfg.Remotes["origin"]
	wantURLs := []string{"https://github.com/owner/repo.git"}
	if !reflect.DeepEqual(rem.URLs, wantURLs) {
		t.Errorf("clean remote URLs = %v, want %v", rem.URLs, wantURLs)
	}
}

func TestGetPushRefSpecs(t *testing.T) {
	t.Parallel()

	repo, err := git.Init(memory.NewStorage(), nil)
	if err != nil {
		t.Fatalf("init repository: %v", err)
	}

	hash := plumbing.NewHash("1111111111111111111111111111111111111111")
	for _, name := range []plumbing.ReferenceName{
		plumbing.NewBranchReferenceName("master"),
		plumbing.NewBranchReferenceName("feature"),
		plumbing.NewTagReferenceName("v1.0.0"),
		plumbing.NewRemoteReferenceName("origin", "feature"),
	} {
		if err := repo.Storer.SetReference(plumbing.NewHashReference(name, hash)); err != nil {
			t.Fatalf("set reference %s: %v", name, err)
		}
	}

	refspecs, err := getPushRefSpecs(repo, plumbing.NewBranchReferenceName("master"))
	if err != nil {
		t.Fatalf("getPushRefSpecs() error: %v", err)
	}

	want := map[string]bool{
		"refs/heads/feature:refs/heads/feature": false,
		"refs/tags/v1.0.0:refs/tags/v1.0.0":     false,
	}
	for _, refspec := range refspecs {
		key := refspec.String()
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected refspec %q", key)
		}
		want[key] = true
	}
	for refspec, found := range want {
		if !found {
			t.Errorf("missing refspec %q", refspec)
		}
	}
}

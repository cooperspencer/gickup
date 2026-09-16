package gitcmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func expectedAuthEnv(username, password string) []string {
	return []string{
		"GICKUP_GIT_USERNAME=" + username,
		"GICKUP_GIT_PASSWORD=" + password,
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=credential.helper",
		"GIT_CONFIG_VALUE_0=",
		"GIT_CONFIG_KEY_1=credential.helper",
		"GIT_CONFIG_VALUE_1=" + envAuthHelper,
	}
}

func TestAuth_Env_Nil(t *testing.T) {
	t.Parallel()

	var auth *Auth
	if got := auth.Env(); got != nil {
		t.Fatalf("expected nil env for nil Auth, got %v", got)
	}
}

func TestAuth_Env_Empty(t *testing.T) {
	t.Parallel()

	auth := &Auth{Username: "", Password: ""}
	if got := auth.Env(); got != nil {
		t.Fatalf("expected nil env for empty Auth, got %v", got)
	}
}

func TestAuth_Env_GitHubApp(t *testing.T) {
	t.Parallel()

	auth := &Auth{
		Username: "x-access-token",
		Password: "test-token",
	}

	env := auth.Env()
	expectedEnv := expectedAuthEnv("x-access-token", "test-token")

	if !reflect.DeepEqual(env, expectedEnv) {
		t.Fatalf("auth.Env() = %v, want %v", env, expectedEnv)
	}
}

func TestAuth_Env_UserPassword(t *testing.T) {
	t.Parallel()

	auth := &Auth{
		Username: "octocat",
		Password: "secretpassword456",
	}

	env := auth.Env()
	expectedEnv := expectedAuthEnv("octocat", "secretpassword456")

	if !reflect.DeepEqual(env, expectedEnv) {
		t.Fatalf("auth.Env() = %v, want %v", env, expectedEnv)
	}
}

func TestGitCmd_Command_WithoutAuth(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}
	cmd := g.Command(context.Background(), nil, "status")

	if cmd.Path != "git" && !strings.HasSuffix(cmd.Path, "git") && !strings.HasSuffix(cmd.Path, "git.exe") {
		t.Errorf("cmd.Path = %q, want 'git'", cmd.Path)
	}
	if !reflect.DeepEqual(cmd.Args, []string{"git", "status"}) {
		t.Errorf("cmd.Args = %v, want ['git', 'status']", cmd.Args)
	}
	if cmd.Env != nil {
		t.Errorf("cmd.Env for nil Auth = %v, want nil", cmd.Env)
	}
}

func TestGitCmd_Command_WithAuth(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}
	auth := &Auth{Username: "x-access-token", Password: "test-token"}
	cmd := g.Command(context.Background(), auth, "clone", "https://github.com/owner/repo.git", "/tmp/repo")

	if cmd.Env == nil {
		t.Fatal("cmd.Env is nil, want environment variables with auth config")
	}

	present := make(map[string]bool, len(cmd.Env))
	for _, e := range cmd.Env {
		present[e] = true
	}
	for _, want := range expectedAuthEnv("x-access-token", "test-token") {
		if !present[want] {
			t.Errorf("cmd.Env missing %q, env = %v", want, cmd.Env)
		}
	}

	// Verify the URL in cmd.Args does NOT contain token
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "test-token") {
			t.Errorf("token leaked into command arguments: %q", arg)
		}
	}
}

func TestGitCmd_CloneCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		reponame string
		bare     bool
		mirror   bool
		auth     *Auth
		wantArgs []string
		wantAuth bool
	}{
		{
			name:     "regular clone without auth",
			url:      "https://github.com/owner/repo.git",
			reponame: "/tmp/repo",
			bare:     false,
			mirror:   false,
			auth:     nil,
			wantArgs: []string{"git", "clone", "https://github.com/owner/repo.git", "/tmp/repo"},
			wantAuth: false,
		},
		{
			name:     "bare clone with token",
			url:      "https://github.com/owner/repo.git",
			reponame: "/tmp/repo.git",
			bare:     true,
			mirror:   false,
			auth:     &Auth{Username: "x-access-token", Password: "my-token"},
			wantArgs: []string{"git", "clone", "https://github.com/owner/repo.git", "/tmp/repo.git", "--bare"},
			wantAuth: true,
		},
		{
			name:     "mirror clone with user/pass",
			url:      "https://github.com/owner/repo.git",
			reponame: "/tmp/repo.git",
			bare:     true,
			mirror:   true,
			auth:     &Auth{Username: "user", Password: "pass"},
			wantArgs: []string{"git", "clone", "https://github.com/owner/repo.git", "/tmp/repo.git", "--bare", "--mirror"},
			wantAuth: true,
		},
	}

	g := GitCmd{CMD: "git"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"clone", tt.url, tt.reponame}
			if tt.bare {
				args = append(args, "--bare")
			}
			if tt.mirror {
				args = append(args, "--mirror")
			}
			cmd := g.Command(context.Background(), tt.auth, args...)

			if !reflect.DeepEqual(cmd.Args, tt.wantArgs) {
				t.Errorf("cmd.Args = %v, want %v", cmd.Args, tt.wantArgs)
			}

			if tt.wantAuth && cmd.Env == nil {
				t.Error("cmd.Env is nil, want auth environment")
			}
			if !tt.wantAuth && cmd.Env != nil {
				t.Errorf("cmd.Env = %v, want nil", cmd.Env)
			}

			for _, arg := range cmd.Args {
				if strings.Contains(arg, "@") {
					t.Errorf("URL in arguments contains credentials: %s", arg)
				}
			}
		})
	}
}

func TestGitCmd_PullCommand(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}
	auth := &Auth{Username: "x-access-token", Password: "test-token"}

	// Bare/mirror fetch
	bareArgs := []string{"-C", "/tmp/repo.git", "fetch", "--all"}
	bareCmd := g.Command(context.Background(), auth, bareArgs...)
	wantBareArgs := []string{"git", "-C", "/tmp/repo.git", "fetch", "--all"}
	if !reflect.DeepEqual(bareCmd.Args, wantBareArgs) {
		t.Errorf("bareCmd.Args = %v, want %v", bareCmd.Args, wantBareArgs)
	}
	if bareCmd.Env == nil {
		t.Error("bareCmd.Env is nil, want auth environment")
	}

	// Non-bare pull
	pullArgs := []string{"-C", "/tmp/repo", "pull", "--all"}
	pullCmd := g.Command(context.Background(), auth, pullArgs...)
	wantPullArgs := []string{"git", "-C", "/tmp/repo", "pull", "--all"}
	if !reflect.DeepEqual(pullCmd.Args, wantPullArgs) {
		t.Errorf("pullCmd.Args = %v, want %v", pullCmd.Args, wantPullArgs)
	}
	if pullCmd.Env == nil {
		t.Error("pullCmd.Env is nil, want auth environment")
	}
}

func TestGitCmd_LFSFetchCommand(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}
	auth := &Auth{Username: "x-access-token", Password: "test-token"}

	lfsArgs := []string{"-C", "/tmp/repo.git", "lfs", "fetch", "--all"}
	cmd := g.Command(context.Background(), auth, lfsArgs...)
	wantArgs := []string{"git", "-C", "/tmp/repo.git", "lfs", "fetch", "--all"}

	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
	}
	if cmd.Env == nil {
		t.Fatal("cmd.Env is nil, want auth environment for LFS fetch")
	}

	foundHelper := false
	for _, e := range cmd.Env {
		if e == "GIT_CONFIG_VALUE_1="+envAuthHelper {
			foundHelper = true
		}
		// git lfs would add an extra header to object downloads next to the
		// Authorization header from the LFS batch response.
		if strings.Contains(e, "http.extraHeader") {
			t.Errorf("cmd.Env sets http.extraHeader, which breaks LFS object downloads: %q", e)
		}
	}
	if !foundHelper {
		t.Error("cmd.Env does not contain the credential helper for LFS fetch")
	}
}

func TestGitCmd_CheckoutCommand(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}

	// Checkout must scope the command to the repo path with -C, otherwise
	// git runs against the process's cwd instead of the cloned repo.
	checkoutArgs := []string{"-C", "/tmp/repo", "checkout", "main"}
	cmd := g.Command(context.Background(), nil, checkoutArgs...)
	wantArgs := []string{"git", "-C", "/tmp/repo", "checkout", "main"}

	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestGitCmd_LocalCommandsHaveNoAuth(t *testing.T) {
	t.Parallel()

	g := GitCmd{CMD: "git"}

	// Checkout should have no auth
	checkoutCmd := g.Command(context.Background(), nil, "checkout", "main")
	if checkoutCmd.Env != nil {
		t.Errorf("checkoutCmd.Env = %v, want nil", checkoutCmd.Env)
	}

	// NewRemote should have no auth
	remoteCmd := g.Command(context.Background(), nil, "-C", "/tmp/repo", "remote", "add", "origin", "https://github.com/owner/repo.git")
	if remoteCmd.Env != nil {
		t.Errorf("remoteCmd.Env = %v, want nil", remoteCmd.Env)
	}
}

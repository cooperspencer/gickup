package gitcmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

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
	expectedHeader := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:test-token"))
	expectedEnv := []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.extraHeader",
		"GIT_CONFIG_VALUE_0=" + expectedHeader,
	}

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
	expectedHeader := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("octocat:secretpassword456"))
	expectedEnv := []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.extraHeader",
		"GIT_CONFIG_VALUE_0=" + expectedHeader,
	}

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

	hasConfigCount := false
	hasConfigKey := false
	hasConfigValue := false

	expectedValue := fmt.Sprintf("GIT_CONFIG_VALUE_0=Authorization: Basic %s",
		base64.StdEncoding.EncodeToString([]byte("x-access-token:test-token")))

	for _, e := range cmd.Env {
		if e == "GIT_CONFIG_COUNT=1" {
			hasConfigCount = true
		}
		if e == "GIT_CONFIG_KEY_0=http.extraHeader" {
			hasConfigKey = true
		}
		if e == expectedValue {
			hasConfigValue = true
		}
	}

	if !hasConfigCount || !hasConfigKey || !hasConfigValue {
		t.Errorf("cmd.Env missing required GIT_CONFIG variables, env = %v", cmd.Env)
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

	foundExtraHeader := false
	for _, e := range cmd.Env {
		if e == "GIT_CONFIG_KEY_0=http.extraHeader" {
			foundExtraHeader = true
		}
	}
	if !foundExtraHeader {
		t.Error("cmd.Env does not contain GIT_CONFIG_KEY_0=http.extraHeader for LFS fetch")
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

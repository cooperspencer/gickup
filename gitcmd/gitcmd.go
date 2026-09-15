package gitcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Auth struct {
	Username string
	Password string
}

// envAuthHelper answers git's credential "get" request from the environment,
// so credentials never end up in URLs, command arguments or git config files.
const envAuthHelper = `!f() { test "$1" = get || return 0; echo "username=${GICKUP_GIT_USERNAME}"; echo "password=${GICKUP_GIT_PASSWORD}"; }; f`

// Env passes the credentials through a credential helper instead of
// http.extraHeader. git lfs adds extra headers to object downloads as well,
// where the LFS server already provides its own Authorization header, and
// servers like gitlab.com reject the duplicate header with 400 Bad Request.
func (a *Auth) Env() []string {
	if a == nil || (a.Username == "" && a.Password == "") {
		return nil
	}
	return []string{
		"GICKUP_GIT_USERNAME=" + a.Username,
		"GICKUP_GIT_PASSWORD=" + a.Password,
		"GIT_CONFIG_COUNT=2",
		// The empty value resets helpers from system or global config,
		// so no other helper stores these credentials.
		"GIT_CONFIG_KEY_0=credential.helper",
		"GIT_CONFIG_VALUE_0=",
		"GIT_CONFIG_KEY_1=credential.helper",
		"GIT_CONFIG_VALUE_1=" + envAuthHelper,
	}
}

type GitCmd struct {
	CMD string
}

func New() (GitCmd, error) {
	cmd := exec.CommandContext(context.Background(), "git", "lfs")
	err := cmd.Run()
	if err != nil {
		return GitCmd{}, errors.New("git lfs is not installed")
	}

	return GitCmd{CMD: "git"}, nil
}

func (g GitCmd) Command(ctx context.Context, auth *Auth, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, g.CMD, args...)
	if authEnv := auth.Env(); len(authEnv) > 0 {
		cmd.Env = append(os.Environ(), authEnv...)
	}
	return cmd
}

func (g GitCmd) Clone(url, reponame string, bare bool, mirror bool, auth *Auth) error {
	args := []string{"clone", url, reponame}
	if bare {
		args = append(args, "--bare")
	}
	if mirror {
		args = append(args, "--mirror")
	}
	cmd := g.Command(context.Background(), auth, args...)
	return cmd.Run()
}

func (g GitCmd) Pull(bare bool, mirror bool, repopath string, auth *Auth) error {
	var args []string
	if bare || mirror {
		args = []string{"-C", repopath, "fetch", "--all"}
	} else {
		args = []string{"-C", repopath, "pull", "--all"}
	}
	cmd := g.Command(context.Background(), auth, args...)
	return cmd.Run()
}

func (g GitCmd) Fetch(path string, auth *Auth) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "fetch", "--all", "--tags"}
	cmd := g.Command(context.Background(), auth, args...)
	return cmd.Run()
}

func (g GitCmd) LFSFetch(path string, auth *Auth) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "lfs", "fetch", "--all"}
	cmd := g.Command(context.Background(), auth, args...)
	return cmd.Run()
}

func (g GitCmd) MirrorPull(path string, auth *Auth) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "pull", "--all", "--tags"}
	cmd := g.Command(context.Background(), auth, args...)
	return cmd.Run()
}

func (g GitCmd) NewRemote(name, url, path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "remote", "add", name, url}
	cmd := g.Command(context.Background(), nil, args...)

	return cmd.Run()
}

func (g GitCmd) Push(path, remote string, auth *Auth) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "push", "--all", remote}
	cmd := g.Command(context.Background(), auth, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%s", strings.TrimSuffix(string(output), "\n"))
		}
	}

	return err
}

func (g GitCmd) Checkout(path, branch string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "checkout", branch}
	cmd := g.Command(context.Background(), nil, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%s", strings.TrimSuffix(string(output), "\n"))
		}
	}

	return err
}

func (g GitCmd) SSHPush(path, remote, key string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "push", "--all", remote}
	cmd := g.Command(context.Background(), nil, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s", key))

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%s", strings.TrimSuffix(string(output), "\n"))
		}
	}

	return err
}

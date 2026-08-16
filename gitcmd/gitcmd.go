package gitcmd

import (
	"context"
	"encoding/base64"
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

func (a *Auth) Env() []string {
	if a == nil || (a.Username == "" && a.Password == "") {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", a.Username, a.Password)))
	return []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.extraHeader",
		fmt.Sprintf("GIT_CONFIG_VALUE_0=Authorization: Basic %s", encoded),
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
	args := []string{"checkout", branch}
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

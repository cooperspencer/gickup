package gitcmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GitCmd struct {
	CMD string
}

func New() (GitCmd, error) {
	cmd := exec.Command("git", "lfs")
	err := cmd.Run()
	if err != nil {
		return GitCmd{}, errors.New("git lfs is not installed")
	}

	return GitCmd{CMD: "git"}, nil
}

func (g GitCmd) Clone(url, path string, bare bool) error {
	cmd := exec.Command(g.CMD, "clone", url, path)
	if bare {
		cmd.Args = append(cmd.Args, "--bare")
	}
	return cmd.Run()
}

func (g GitCmd) Pull(bare bool) error {
	var args = []string{}
	if bare {
		args = []string{"fetch", "--all"}
	} else {
		args = []string{"pull", "--all"}
	}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) Fetch(path string) error {
	currentpath, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(currentpath)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	args := []string{"fetch", "--all", "--tags"}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) MirrorPull(path string) error {
	currentpath, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(currentpath)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	args := []string{"pull", "--all", "--tags"}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) NewRemote(name, url, path string) error {
	currentpath, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(currentpath)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	args := []string{"remote", "add", name, url}
	cmd := exec.Command(g.CMD, args...)

	return cmd.Run()
}

func (g GitCmd) Push(path, remote string) error {
	currentpath, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(currentpath)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	args := []string{"push", "--all", remote}
	cmd := exec.Command(g.CMD, args...)

	output, _ := cmd.CombinedOutput()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(strings.TrimSuffix(string(output), "\n"))
	}

	return nil
}

func (g GitCmd) SSHPush(path, remote, key string) error {
	os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s", key))

	return g.Push(path, remote)
}

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
	cmd := exec.Command("git", "--help")
	err := cmd.Run()
	if err != nil {
		return GitCmd{}, errors.New("git is not installed")
	}

	cmd = exec.Command("git", "lfs")
	err = cmd.Run()
	if err != nil {
		return GitCmd{}, errors.New("git lfs is not installed")
	}

	return GitCmd{CMD: "git"}, nil
}

func (g GitCmd) Clone(url, reponame string, bare bool, mirror bool) error {
	cmd := exec.Command(g.CMD, "clone", url, reponame)
	if bare {
		cmd.Args = append(cmd.Args, "--bare")
	}
	if mirror {
		cmd.Args = append(cmd.Args, "--mirror")
	}
	return cmd.Run()
}

func (g GitCmd) Pull(bare bool, mirror bool, repopath string) error {
	var args = []string{}
	if bare || mirror {
		args = []string{"-C", repopath, "fetch", "--all"}
	} else {
		args = []string{"-C", repopath, "pull", "--all"}
	}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) Fetch(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "fetch", "--all", "--tags"}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) LFSFetch(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "lfs", "fetch", "--all"}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) MirrorPull(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "pull", "--all", "--tags"}
	cmd := exec.Command(g.CMD, args...)
	return cmd.Run()
}

func (g GitCmd) NewRemote(name, url, path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "remote", "add", name, url}
	cmd := exec.Command(g.CMD, args...)

	return cmd.Run()
}

func (g GitCmd) Push(path, remote string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	args := []string{"-C", path, "push", "--all", remote}
	cmd := exec.Command(g.CMD, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
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
	cmd := exec.Command(g.CMD, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("%s", strings.TrimSuffix(string(output), "\n"))
		}
	}

	return err
}

func (g GitCmd) SSHPush(path, remote, key string) error {
	err := os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s", key))
	if err != nil {
		return err
	}

	return g.Push(path, remote)
}

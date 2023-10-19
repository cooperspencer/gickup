package gitcmd

import (
	"errors"
	"os/exec"
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
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (g GitCmd) Pull(bare bool) error {
	var args = []string{}
	if bare {
		args = []string{"fetch", "--all"}
	} else {
		args = []string{"pull"}
	}
	cmd := exec.Command(g.CMD, args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

package whatever

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestGetBuildsRemoteRepo(t *testing.T) {
	t.Parallel()

	conf := &types.Conf{
		Source: types.Source{
			Any: []types.GenRepo{{URL: "https://example.com/org/repo.git"}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	repo := repos[0]
	if repo.Name != "repo" || repo.Hoster != "example.com" || repo.Owner != "git" {
		t.Fatalf("unexpected repo: %#v", repo)
	}
}

func TestGetBuildsLocalRepo(t *testing.T) {
	t.Parallel()

	localPath := filepath.Join(t.TempDir(), "repo.git")
	if err := os.MkdirAll(localPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	conf := &types.Conf{
		Source: types.Source{
			Any: []types.GenRepo{{URL: localPath, Username: "alice"}},
		},
	}

	repos, ran := Get(conf)
	if !ran {
		t.Fatal("expected adapter to run")
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	repo := repos[0]
	if repo.Name != "repo" || repo.Hoster != "local" || repo.Owner != "alice" {
		t.Fatalf("unexpected repo: %#v", repo)
	}
}

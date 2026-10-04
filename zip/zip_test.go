package zip

import (
	archivezip "archive/zip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZipCreatesArchiveAndRemovesSources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoDir := filepath.Join(root, "repo")
	issuesDir := filepath.Join(root, "repo.issues")

	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(issuesDir, 0o755); err != nil {
		t.Fatalf("mkdir issues: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, "refs", "heads"), 0o755); err != nil {
		t.Fatalf("mkdir empty directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("repo data"), 0o644); err != nil {
		t.Fatalf("write repo file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuesDir, "1.json"), []byte(`{"id":1}`), 0o644); err != nil {
		t.Fatalf("write issue file: %v", err)
	}

	if err := Zip(repoDir, []string{repoDir, issuesDir}); err != nil {
		t.Fatalf("Zip() error = %v", err)
	}

	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Fatalf("expected repo dir removal, got err=%v", err)
	}
	if _, err := os.Stat(issuesDir); !os.IsNotExist(err) {
		t.Fatalf("expected issues dir removal, got err=%v", err)
	}

	archivePath := repoDir + ".zip"
	reader, err := archivezip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	entries := map[string]string{}
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/") && !file.FileInfo().IsDir() {
			t.Fatalf("entry %q is not marked as a directory", file.Name)
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", file.Name, err)
		}

		data, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", file.Name, err)
		}

		entries[file.Name] = string(data)
	}

	if entries["repo/README.md"] != "repo data" {
		t.Fatalf("unexpected repo entry contents: %#v", entries)
	}

	if entries["repo.issues/1.json"] != `{"id":1}` {
		t.Fatalf("unexpected issue entry contents: %#v", entries)
	}
	for _, name := range []string{"repo/", "repo/refs/", "repo/refs/heads/", "repo.issues/"} {
		if _, ok := entries[name]; !ok {
			t.Errorf("missing directory entry %q", name)
		}
	}
}

func TestZipRestoresBareRepositoryWithPackedRefs(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for the restore test")
	}
	root := t.TempDir()
	runGit := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(context.TODO(), "git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	source := filepath.Join(root, "source")
	repo := filepath.Join(root, "repo.git")
	runGit("init", "--initial-branch=main", source)
	runGit("-C", source, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "test")
	runGit("-C", source, "tag", "v1")
	runGit("clone", "--mirror", source, repo)
	runGit("-C", repo, "pack-refs", "--all", "--prune")
	wantRefs := runGit("-C", repo, "show-ref")
	for _, name := range []string{"heads", "tags"} {
		entries, err := os.ReadDir(filepath.Join(repo, "refs", name))
		if err != nil || len(entries) != 0 {
			t.Fatalf("expected empty refs/%s before archiving, got %v, %v", name, entries, err)
		}
	}
	if err := Zip(repo, []string{repo}); err != nil {
		t.Fatal(err)
	}
	reader, err := archivezip.OpenReader(repo + ".zip")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	restoreDir := filepath.Join(root, "restored")
	for _, entry := range reader.File {
		dest := filepath.Join(restoreDir, filepath.FromSlash(entry.Name))
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		r, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	restored := filepath.Join(restoreDir, "repo.git")
	if got := runGit("-C", restored, "rev-parse", "--is-bare-repository"); got != "true" {
		t.Fatalf("restored repository is not bare: %s", got)
	}
	runGit("-C", restored, "fsck", "--full")
	if got := runGit("-C", restored, "show-ref"); got != wantRefs {
		t.Fatalf("restored refs = %q, want %q", got, wantRefs)
	}
}

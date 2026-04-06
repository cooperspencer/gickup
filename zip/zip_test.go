package zip

import (
	archivezip "archive/zip"
	"io"
	"os"
	"path/filepath"
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
}

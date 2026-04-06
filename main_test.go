package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTildeReplacement_NoAction(t *testing.T) {
	t.Parallel()

	if path := "/boop"; substituteHomeForTildeInPath(path) != path {
		t.Error("Altered path when no alteration was expected")
	}
}

func TestTildeReplacement_TildeOnly(t *testing.T) {
	t.Parallel()

	if path := "~"; substituteHomeForTildeInPath(path) == path {
		t.Error("Path unaltered when alteration was expected")
	}
}

func TestTildeReplacement_TildeDir(t *testing.T) {
	t.Parallel()

	path := "~/boop"
	actual := substituteHomeForTildeInPath(path)
	if strings.HasPrefix(actual, "~") {
		t.Error("Altered path still contains ~")
	}

	if !strings.HasSuffix(actual, "boop") {
		t.Error("Altered path does not end with directory to be retained")
	}
}

func TestReadConfigFile_InheritsPushConfigsAndExpandsHome(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "conf.yml")
	config := `destination:
  local:
    - path: "~/primary"
metrics:
  push:
    ntfy:
      - url: "https://ntfy.sh/topic"
        token: "secret"
---
destination:
  local:
    - path: "/tmp/secondary"
`

	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	confs := readConfigFile(configPath)
	if len(confs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(confs))
	}

	if got := confs[0].Destination.Local[0].Path; strings.HasPrefix(got, "~") {
		t.Fatalf("expected home expansion, got %q", got)
	}

	if diff := len(confs[1].Metrics.PushConfigs.Ntfy); diff != 1 {
		t.Fatalf("expected inherited ntfy config, got %d entries", diff)
	}

	if got := confs[1].Metrics.PushConfigs.Ntfy[0]; got.Url != "https://ntfy.sh/topic" || got.Token != "secret" {
		t.Fatalf("unexpected inherited push config: %+v", got)
	}

	if len(confs[0].Metrics.PushConfigs.Ntfy) != 1 {
		t.Fatal("expected initial push config to be populated")
	}
}

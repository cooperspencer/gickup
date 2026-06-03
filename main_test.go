package main

import (
	"os"
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
	f, err := os.CreateTemp(t.TempDir(), "gickup-test-*.yml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	configPath := f.Name()
	if _, err := f.WriteString(config); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()

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

func TestReadConfigFile_S3UseStaticCredsTrue(t *testing.T) {
	t.Parallel()

	config := `destination:
  s3:
    - bucket: "my-bucket"
      endpoint: "s3.example.com"
      use_static_creds: true
      accesskey: "AKID"
      secretkey: "SECRET"
`
	f, err := os.CreateTemp(t.TempDir(), "gickup-test-*.yml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	configPath := f.Name()
	if _, err := f.WriteString(config); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()

	confs := readConfigFile(configPath)
	if len(confs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(confs))
	}

	s3 := confs[0].Destination.S3
	if len(s3) != 1 {
		t.Fatalf("expected 1 S3 destination, got %d", len(s3))
	}

	if !s3[0].UseStaticCreds {
		t.Fatal("expected UseStaticCreds to be true")
	}
}

func TestReadConfigFile_S3UseStaticCredsFalse(t *testing.T) {
	t.Parallel()

	config := `destination:
  s3:
    - bucket: "my-bucket"
      endpoint: "s3.example.com"
      use_static_creds: false
`
	f, err := os.CreateTemp(t.TempDir(), "gickup-test-*.yml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	configPath := f.Name()
	if _, err := f.WriteString(config); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()

	confs := readConfigFile(configPath)
	if len(confs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(confs))
	}

	s3 := confs[0].Destination.S3
	if len(s3) != 1 {
		t.Fatalf("expected 1 S3 destination, got %d", len(s3))
	}

	if s3[0].UseStaticCreds {
		t.Fatal("expected UseStaticCreds to be false")
	}
}

func TestReadConfigFile_S3UseStaticCredsResolvesEnvVars(t *testing.T) {
	t.Setenv("TEST_S3_ACCESS", "resolved-access")
	t.Setenv("TEST_S3_SECRET", "resolved-secret")

	config := `destination:
  s3:
    - bucket: "my-bucket"
      endpoint: "s3.example.com"
      use_static_creds: true
      accesskey: "TEST_S3_ACCESS"
      secretkey: "TEST_S3_SECRET"
`
	f, err := os.CreateTemp(t.TempDir(), "gickup-test-*.yml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	configPath := f.Name()
	if _, err := f.WriteString(config); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()

	confs := readConfigFile(configPath)
	s3 := confs[0].Destination.S3[0]

	// Simulate the key resolution that backup() does when UseStaticCreds is true
	if !s3.UseStaticCreds {
		t.Fatal("expected UseStaticCreds to be true")
	}

	accessKey, err := s3.GetKey(s3.AccessKey)
	if err != nil {
		t.Fatalf("GetKey(accesskey) error = %v", err)
	}

	secretKey, err := s3.GetKey(s3.SecretKey)
	if err != nil {
		t.Fatalf("GetKey(secretkey) error = %v", err)
	}

	if accessKey != "resolved-access" {
		t.Fatalf("accesskey = %q, want resolved-access", accessKey)
	}

	if secretKey != "resolved-secret" {
		t.Fatalf("secretkey = %q, want resolved-secret", secretKey)
	}
}

func TestReadConfigFile_S3UseStaticCredsAbsentSkipsKeyResolution(t *testing.T) {
	t.Parallel()

	config := `destination:
  s3:
    - bucket: "my-bucket"
      endpoint: "s3.example.com"
`
	f, err := os.CreateTemp(t.TempDir(), "gickup-test-*.yml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	configPath := f.Name()
	if _, err := f.WriteString(config); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()

	confs := readConfigFile(configPath)
	s3 := confs[0].Destination.S3[0]

	// When UseStaticCreds is false (zero value), backup() skips key resolution entirely
	if s3.UseStaticCreds {
		t.Fatal("expected UseStaticCreds to be false when absent from config")
	}
}

package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfCronMissing(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: ""}

	if !conf.MissingCronSpec() {
		t.Error("Conf passed empty cron spec doesn't think that it's missing")
	}
}

func TestParseValidCron(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: "0 0 * * *"}

	if !conf.HasValidCronSpec() {
		t.Error("Valid cron spec parsed invalidly")
	}
}

func TestParseInvalidCron(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: "redshirt"}

	if conf.HasValidCronSpec() {
		t.Error("Invalid cron spec parsed validly")
	}
}

func TestDestinationCount(t *testing.T) {
	t.Parallel()

	dest := Destination{
		Gitlab:    []GenRepo{{}, {}},
		Local:     []Local{{}},
		Github:    []GenRepo{{}},
		Gitea:     []GenRepo{{}},
		Gogs:      []GenRepo{{}},
		OneDev:    []GenRepo{{}},
		Sourcehut: []GenRepo{{}},
		S3:        []S3Repo{{}},
		AzureBlob: []AzureBlob{{}},
	}

	if got, want := dest.Count(), 10; got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}
}

func TestSourceCount(t *testing.T) {
	t.Parallel()

	source := Source{
		Gogs:      []GenRepo{{}},
		Gitlab:    []GenRepo{{}},
		Github:    []GenRepo{{}, {}},
		Gitea:     []GenRepo{{}},
		BitBucket: []GenRepo{{}},
		OneDev:    []GenRepo{{}},
		Sourcehut: []GenRepo{{}},
		Any:       []GenRepo{{}},
	}

	if got, want := source.Count(), 9; got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}
}

func TestPushConfigResolveToken(t *testing.T) {
	t.Setenv("PUSH_PASSWORD", "password")
	t.Setenv("PUSH_TOKEN", "token")

	config := PushConfig{Password: "PUSH_PASSWORD", Token: "PUSH_TOKEN"}
	config.ResolveToken()

	if config.Password != "password" {
		t.Fatalf("password = %q, want resolved env value", config.Password)
	}

	if config.Token != "token" {
		t.Fatalf("token = %q, want resolved env value", config.Token)
	}
}

func TestResolveFallsBackToLiteral(t *testing.T) {
	t.Parallel()

	if got := resolve("literal"); got != "literal" {
		t.Fatalf("resolve() = %q, want literal", got)
	}
}

func TestCheckAllValuesOrNone(t *testing.T) {
	t.Parallel()

	if !CheckAllValuesOrNone("prometheus", map[string]string{"listenaddr": ":8080", "endpoint": "/metrics"}) {
		t.Fatal("expected all populated values to pass")
	}

	if CheckAllValuesOrNone("prometheus", map[string]string{"listenaddr": ":8080", "endpoint": ""}) {
		t.Fatal("expected missing values to fail")
	}
}

func TestGetNextRunMissingCron(t *testing.T) {
	t.Parallel()

	conf := Conf{}

	if _, err := conf.GetNextRun(); err == nil {
		t.Fatal("expected error for missing cron")
	}
}

func TestGetNextRunValidCron(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: "0 0 * * *"}

	next, err := conf.GetNextRun()
	if err != nil {
		t.Fatalf("GetNextRun() error = %v", err)
	}

	if next == nil {
		t.Fatal("expected next run time")
	}

	if next.Before(time.Now()) {
		t.Fatalf("expected next run in the future, got %v", next)
	}
}

func TestFilterParseDuration(t *testing.T) {
	t.Parallel()

	filter := Filter{LastActivityString: "1d2h30m"}

	if err := filter.ParseDuration(); err != nil {
		t.Fatalf("ParseDuration() error = %v", err)
	}

	want := 26*time.Hour + 30*time.Minute
	if diff := filter.LastActivityDuration - want; diff < -time.Minute || diff > time.Minute {
		t.Fatalf("duration = %v, want about %v", filter.LastActivityDuration, want)
	}
}

func TestFilterParseDurationInvalid(t *testing.T) {
	t.Parallel()

	filter := Filter{LastActivityString: "nonsense"}

	if err := filter.ParseDuration(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestResolveTokenPrefersEnvironment(t *testing.T) {
	t.Setenv("GENERIC_TOKEN", "resolved-token")

	got, err := resolveToken("GENERIC_TOKEN", "")
	if err != nil {
		t.Fatalf("resolveToken() error = %v", err)
	}

	if got != "resolved-token" {
		t.Fatalf("resolveToken() = %q, want env value", got)
	}
}

func TestResolveTokenReadsFromFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(path, []byte("file-token\n"), 0o644); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	got, err := resolveToken("", path)
	if err != nil {
		t.Fatalf("resolveToken() error = %v", err)
	}

	if got != "file-token" {
		t.Fatalf("resolveToken() = %q, want file-token", got)
	}
}

func TestResolveTokenWithNoSource(t *testing.T) {
	t.Parallel()

	got, err := resolveToken("", "")
	if err != nil {
		t.Fatalf("resolveToken() error = %v", err)
	}

	if got != "" {
		t.Fatalf("resolveToken() = %q, want empty string", got)
	}
}

func TestGenRepoGetTokenFromEnvironment(t *testing.T) {
	t.Setenv("REPO_TOKEN", "repo-secret")

	if got := (GenRepo{Token: "REPO_TOKEN"}).GetToken(); got != "repo-secret" {
		t.Fatalf("GetToken() = %q, want env value", got)
	}
}

func TestGetHost(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"https://example.com/org/repo": "example.com",
		"http://example.com/org/repo":  "example.com",
		"example.com/org/repo":         "example.com",
	}

	for input, want := range tests {
		if got := GetHost(input); got != want {
			t.Fatalf("GetHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSiteGetValuesSSHURL(t *testing.T) {
	t.Parallel()

	var site Site

	if err := site.GetValues("ssh://git@example.com:2222/org/repo.git"); err != nil {
		t.Fatalf("GetValues() error = %v", err)
	}

	if site.User != "git" || site.URL != "example.com" || site.Port != 2222 {
		t.Fatalf("unexpected site values: %+v", site)
	}
}

func TestSiteGetValuesScpStyle(t *testing.T) {
	t.Parallel()

	var site Site

	if err := site.GetValues("git@example.com:org/repo.git"); err != nil {
		t.Fatalf("GetValues() error = %v", err)
	}

	if site.User != "git" || site.URL != "example.com" || site.Port != 22 {
		t.Fatalf("unexpected site values: %+v", site)
	}
}

func TestGetMap(t *testing.T) {
	t.Parallel()

	got := GetMap([]string{"repo-a", "repo-b"})
	if !got["repo-a"] || !got["repo-b"] || len(got) != 2 {
		t.Fatalf("unexpected map contents: %#v", got)
	}
}

func TestS3RepoGetKey(t *testing.T) {
	t.Setenv("S3_KEY", "resolved-s3-key")

	if got, err := (S3Repo{}).GetKey(""); err == nil || got != "" {
		t.Fatalf("expected empty key error, got value=%q err=%v", got, err)
	}

	if got, err := (S3Repo{}).GetKey("S3_KEY"); err != nil || got != "resolved-s3-key" {
		t.Fatalf("GetKey() = %q, %v, want env value", got, err)
	}

	if got, err := (S3Repo{}).GetKey("literal-key"); err != nil || got != "literal-key" {
		t.Fatalf("GetKey() = %q, %v, want literal-key", got, err)
	}
}

func TestResolveTokenPreservesLiteralToken(t *testing.T) {
	t.Parallel()

	got, err := resolveToken("literal-token", "")
	if err != nil {
		t.Fatalf("resolveToken() error = %v", err)
	}

	if got != "literal-token" {
		t.Fatalf("resolveToken() = %q, want literal-token", got)
	}
}

func TestResolveTrimsNothingFromLiteral(t *testing.T) {
	t.Parallel()

	if got := resolve("already-set"); !strings.EqualFold(got, "already-set") {
		t.Fatalf("resolve() = %q, want already-set", got)
	}
}

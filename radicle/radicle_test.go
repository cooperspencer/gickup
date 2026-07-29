package radicle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestHome(t *testing.T) {
	// Home delegates to `rad self --home`; stub rad and check we return what
	// it reports, verbatim.
	bin := t.TempDir()
	script := "#!/bin/sh\n[ \"$1 $2\" = \"self --home\" ] && echo /stub/radhome\n"
	if err := os.WriteFile(filepath.Join(bin, "rad"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", bin, os.PathListSeparator, os.Getenv("PATH")))

	home, err := Home()
	if err != nil {
		t.Fatalf("Home() error: %v", err)
	}
	if home != "/stub/radhome" {
		t.Errorf("expected /stub/radhome from rad self --home, got %q", home)
	}
}

func TestVisibilityFlag(t *testing.T) {
	t.Parallel()

	private := types.Repo{Private: true}
	public := types.Repo{Private: false}

	for _, tc := range []struct {
		visibility string
		repo       types.Repo
		expected   string
	}{
		{"public", private, "--public"},
		{"private", public, "--private"},
		{"", private, "--private"},
		{"", public, "--public"},
		{"source", private, "--private"},
		{"source", public, "--public"},
	} {
		flag, err := visibilityFlag(types.Radicle{Visibility: tc.visibility}, tc.repo)
		if err != nil {
			t.Errorf("visibility %q: %v", tc.visibility, err)
		}

		if flag != tc.expected {
			t.Errorf("visibility %q private=%v: expected %s, got %s", tc.visibility, tc.repo.Private, tc.expected, flag)
		}
	}

	if _, err := visibilityFlag(types.Radicle{Visibility: "invisible"}, public); err == nil {
		t.Error("expected an error for an invalid visibility")
	}
}

func TestPushArgs(t *testing.T) {
	t.Parallel()

	args := pushArgs(types.Radicle{}, "zRid", "zNid")
	expected := "push rad://zRid/zNid refs/heads/*:refs/heads/* refs/tags/*:refs/tags/*"

	if strings.Join(args, " ") != expected {
		t.Errorf("expected %q, got %q", expected, strings.Join(args, " "))
	}

	args = pushArgs(types.Radicle{Force: true, Prune: true}, "zRid", "zNid")
	expected = "push --prune rad://zRid/zNid +refs/heads/*:refs/heads/* +refs/tags/*:refs/tags/*"

	if strings.Join(args, " ") != expected {
		t.Errorf("expected %q, got %q", expected, strings.Join(args, " "))
	}
}

const testNID = "zTestNodeId"

type fakePayload struct {
	name string
	// source is the upstream URL in the dev.gickup payload; empty simulates
	// a repository that was not created by gickup.
	source string
	// owned repos list this profile among the identity delegates; unowned ones
	// list a foreign delegate, simulating a repo merely seeded from the network.
	owned bool
}

// fakeRad puts a rad stub on PATH answering `rad self --home`, `rad inspect
// <rid> --payload` and `rad inspect <rid> --delegates` per configured rid, and
// creates an (empty) storage directory per rid so payloadsFor can enumerate
// them — letting scanStorage be tested without a radicle installation.
func fakeRad(t *testing.T, radhome string, repos map[string]fakePayload) {
	t.Helper()

	bin := t.TempDir()
	ourDID := "did:key:" + testNID

	var payloadCases, delegateCases string
	for rid, p := range repos {
		gickup := ""
		if p.source != "" {
			gickup = fmt.Sprintf("\"dev.gickup\": {\"source\": \"%s\"}, ", p.source)
		}
		payloadCases += fmt.Sprintf("rad:%s) echo '{%s\"xyz.radicle.project\": {\"name\": \"%s\", \"defaultBranch\": \"main\"}}' ;;\n", rid, gickup, p.name)

		delegate := "did:key:zForeignNode"
		if p.owned {
			delegate = ourDID
		}
		delegateCases += fmt.Sprintf("rad:%s) echo '%s (alias)' ;;\n", rid, delegate)
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1 $2\" = \"self --home\" ]; then echo \"$RAD_HOME\"; exit 0; fi\n" +
		"case \"$3\" in\n" +
		"--payload) case \"$2\" in\n" + payloadCases + "*) echo 'unknown repository' >&2; exit 1 ;;\nesac ;;\n" +
		"--delegates) case \"$2\" in\n" + delegateCases + "*) echo 'unknown repository' >&2; exit 1 ;;\nesac ;;\n" +
		"esac\n"

	if err := os.WriteFile(filepath.Join(bin, "rad"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", fmt.Sprintf("%s%c%s", bin, os.PathListSeparator, os.Getenv("PATH")))

	for rid := range repos {
		if err := os.MkdirAll(filepath.Join(radhome, "storage", rid), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanStorage(t *testing.T) {
	radhome := t.TempDir()
	fakeRad(t, radhome, map[string]fakePayload{
		// same repo name upstream, distinguished purely by recorded source
		"zRepoOne": {"website", "https://github.com/alice/website", true},
		"zRepoTwo": {"website", "https://gitea.local/bob/website", true},
		"zOther":   {"other", "https://github.com/alice/other", true},
		// not created by gickup: no dev.gickup payload
		"zForeign": {"website", "", true},
	})

	t.Setenv("RAD_HOME", radhome)

	rid, err := scanStorage(types.Repo{Name: "website", URL: "https://gitea.local/bob/website"}, testNID, os.Environ())
	if err != nil {
		t.Fatalf("scan storage: %v", err)
	}

	if rid != "zRepoTwo" {
		t.Errorf("expected zRepoTwo, got %s", rid)
	}

	rid, err = scanStorage(types.Repo{Name: "website", URL: "https://github.com/carol/website"}, testNID, os.Environ())
	if err != nil {
		t.Fatalf("scan storage: %v", err)
	}

	if rid != "" {
		t.Errorf("expected no match for an unmirrored upstream, got %s", rid)
	}

	// credentials in the configured URL don't prevent matching the
	// credential-free recorded source
	rid, err = scanStorage(types.Repo{Name: "other", URL: "https://user:token@github.com/alice/other"}, testNID, os.Environ())
	if err != nil {
		t.Fatalf("scan storage: %v", err)
	}

	if rid != "zOther" {
		t.Errorf("expected zOther, got %s", rid)
	}
}

func TestScanStorageAmbiguous(t *testing.T) {
	radhome := t.TempDir()
	url := "https://github.com/alice/same"
	fakeRad(t, radhome, map[string]fakePayload{
		"zRepoOne": {"same", url, true},
		"zRepoTwo": {"same", url, true},
	})

	t.Setenv("RAD_HOME", radhome)

	if _, err := scanStorage(types.Repo{Name: "same", URL: url}, testNID, os.Environ()); err == nil {
		t.Error("expected an error for an ambiguous source match")
	}
}

func TestScanStorageRejectsForgedPayloads(t *testing.T) {
	radhome := t.TempDir()
	victim := "https://github.com/victim/foo"
	fakeRad(t, radhome, map[string]fakePayload{
		// a repo seeded from the network whose crafted identity document
		// claims the victim's source, but whose delegates don't include this
		// profile — the delegate check must reject it
		"zSeeded": {"foo", victim, false},
	})

	t.Setenv("RAD_HOME", radhome)

	rid, err := scanStorage(types.Repo{Name: "foo", URL: victim}, testNID, os.Environ())
	if err != nil {
		t.Fatalf("scan storage: %v", err)
	}

	if rid != "" {
		t.Errorf("forged payload was adopted: %s", rid)
	}
}

func TestMarkerStripsCredentials(t *testing.T) {
	t.Parallel()

	m := marker("https://user:secret-token@github.com/alice/repo.git")

	if strings.Contains(m, "secret-token") || strings.Contains(m, "user") {
		t.Errorf("marker leaks credentials: %s", m)
	}

	if m != "(mirror of https://github.com/alice/repo.git)" {
		t.Errorf("unexpected marker: %s", m)
	}

	// non-URL remotes pass through untouched
	if marker("git@github.com:alice/repo.git") != "(mirror of git@github.com:alice/repo.git)" {
		t.Error("scp-style remote was altered")
	}
}

// Package radicle mirrors repositories into the local storage of a Radicle
// node.
package radicle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
)

var sub zerolog.Logger

// payloadCache memoizes the storage scan so scanStorage inspects the storage
// only once per RAD_HOME, instead of once per mirrored repository. It is
// populated on the first scan and kept in sync as initRepo creates mirrors,
// so this process never re-inspects storage it already knows.
var payloadCache = map[string]map[string]payload{}

// payload is the identity document `payload` section, as returned by `rad inspect --payload`.
type payload struct {
	Project struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		DefaultBranch string `json:"defaultBranch"`
	} `json:"xyz.radicle.project"`
	// Gickup is the provenance extension this package adds to the identity
	// document of every mirror it initializes.
	Gickup struct {
		Source string `json:"source"`
	} `json:"dev.gickup"`
}

// Mirror pushes the repository cloned at tempdir into the Radicle storage
// configured by d, initializing it with `rad init` on first sight. It returns
// the RID of the mirror.
func Mirror(repo types.Repo, dest types.Radicle, tempdir string) (string, error) {
	home, _ := Home()
	sub = logger.CreateSubLogger("stage", "radicle", "home", home)

	if err := probe(); err != nil {
		return "", err
	}

	// rad and git-remote-rad read RAD_HOME and RAD_PASSPHRASE from the
	// environment themselves, so the ambient environment is all they need.
	env := os.Environ()

	nid, err := nodeID(env)
	if err != nil {
		return "", err
	}

	rid, err := scanStorage(repo, nid, env)
	if err != nil {
		return "", err
	}

	if rid == "" {
		rid, err = initRepo(repo, dest, tempdir, env)
		if err != nil {
			return "", err
		}

		sub.Info().
			Str("rid", fmt.Sprintf("rad:%s", rid)).
			Msgf("initialized %s", types.Green(repo.Name))
	} else if dest.Force || dest.Prune {
		if err := fetchMirrorState(tempdir, rid, nid, env); err != nil {
			return "", err
		}
	}

	if err := push(dest, tempdir, rid, nid, env); err != nil {
		return "", err
	}

	// Issues coming soon
	if dest.Issues {
		sub.Warn().Msg("Issue migration to Radicle not yet supported.")
	}

	return rid, nil
}

// probe checks that all binaries required for mirroring to Radicle are available.
func probe() error {
	for _, bin := range []string{"git", "rad", "git-remote-rad"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s is not installed", bin)
		}
	}

	return nil
}

// fetchMirrorState fetches the current refs of the mirror into the temporary
// clone. When upstream history was rewritten or a branch was deleted, the old
// ref targets no longer exist in a fresh clone, and the remote helper then
// fails to re-sign rad/sigrefs after the ref update, leaving the signed refs
// pointing at the old state. Fetching first keeps the old objects resolvable
// so refs and signature update together. Pushes without force and prune are
// fast-forward only, where the old targets are always present, so the fetch
// is skipped.
func fetchMirrorState(tempdir, rid, nid string, env []string) error {
	_, err := run(tempdir, env, "git", "fetch", "--no-tags",
		fmt.Sprintf("rad://%s/%s", rid, nid),
		"+refs/heads/*:refs/gickup/heads/*",
		"+refs/tags/*:refs/gickup/tags/*")

	return err
}

// Home returns the profile's Radicle home as reported by `rad self --home`.
// Propagates error accordingly, so it can be handled at call sites.
func Home() (string, error) {
	return run("", os.Environ(), "rad", "self", "--home")
}

// run executes name with args in dir, returning trimmed stdout. Stderr is
// folded into the returned error.
func run(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}

		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// nodeID returns the node ID of the configured profile, which namespaces the
// push URL. `rad self --nid` is deprecated but still works without a running
// node, so it is used as the fallback.
func nodeID(env []string) (string, error) {
	nid, err := run("", env, "rad", "node", "status", "--only", "nid")
	if err != nil {
		nid, err = run("", env, "rad", "self", "--nid")
		if err != nil {
			return "", err
		}
	}

	if nid == "" {
		return "", fmt.Errorf("couldn't determine the node id of the radicle profile")
	}

	return nid, nil
}

// marker returns the human-readable provenance note that initRepo appends to
// the mirror's description. Any credentials embedded in the URL are stripped, because
// the description becomes part of the signed identity payload that is
// replicated to the network.
func marker(upstream string) string {
	return fmt.Sprintf("(mirror of %s)", stripCredentials(upstream))
}

// stripCredentials removes the userinfo part from a URL, so tokens or
// passwords configured inline (https://user:token@host/...) never leave the
// local configuration. Non-URL remotes (e.g. scp-style ssh) pass through. This
// is useful because the identity document, where this URL is stored, is signed
// and published to the network.
func stripCredentials(upstream string) string {
	if !strings.Contains(upstream, "://") {
		return upstream
	}

	parsed, err := url.Parse(upstream)
	if err != nil || parsed.User == nil {
		return upstream
	}

	parsed.User = nil

	return parsed.String()
}

// scanStorage searches the Radicle storage for the repository that mirrors
// the upstream URL. A candidate must record the URL as its source in the
// dev.gickup `payload` of its identity document, and list this profile among
// that identity's delegates. The second check is what makes adoption
// unforgeable: the source claim lives in the identity document, whose contents
// are only authoritative because its delegates signed them.
func scanStorage(repo types.Repo, nid string, env []string) (string, error) {
	inspected, err := payloadsFor(env)
	if err != nil {
		return "", err
	}

	matches := []string{}
	source := stripCredentials(repo.URL)

	for rid, p := range inspected {
		if p.Gickup.Source == source && isDelegate(rid, nid, env) {
			matches = append(matches, rid)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf(
			"found %d repositories in storage mirroring %s - remove the duplicates so only one remains",
			len(matches), repo.URL)
	}
}

// isDelegate reports whether this profile's key (nid) is a delegate of the
// stored repository's identity. We consider this enough evidence that the
// information we read from the identity doc can be trusted.
func isDelegate(rid, nid string, env []string) bool {
	out, err := run("", env, "rad", "inspect", fmt.Sprintf("rad:%s", rid), "--delegates")
	if err != nil {
		return false
	}

	// Each line looks like "did:key:<nid> (alias)".
	did := "did:key:" + nid
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == did {
			return true
		}
	}

	return false
}

// payloadsFor returns the identity payloads of all repositories in the Radicle
// profile storage. The storage is inspected the first time a given RAD_HOME is asked
// about and the result is cached, so the repeated scanStorage calls across a
// run's repositories don't each re-inspect it.
func payloadsFor(env []string) (map[string]payload, error) {
	home, err := Home()
	if err != nil {
		return nil, err
	}

	if cached, ok := payloadCache[home]; ok {
		return cached, nil
	}

	entries, err := os.ReadDir(filepath.Join(home, "storage"))
	if err != nil {
		if os.IsNotExist(err) {
			payloadCache[home] = map[string]payload{}
			return payloadCache[home], nil
		}

		return nil, err
	}

	current := map[string]payload{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		out, err := run("", env, "rad", "inspect", fmt.Sprintf("rad:%s", entry.Name()), "--payload")
		if err != nil {
			sub.Debug().
				Str("rid", fmt.Sprintf("rad:%s", entry.Name())).
				Msg(err.Error())

			continue
		}

		var p payload
		if err := json.Unmarshal([]byte(out), &p); err != nil {
			continue
		}

		current[entry.Name()] = p
	}

	payloadCache[home] = current

	return current, nil
}

// initRepo runs `rad init` on the temporary clone, creating the repository
// identity in storage, and returns the new RID. The upstream URL is recorded
// in a dev.gickup extension of the identity document `payload`, which is what
// scanStorage matches on, on the next run.
func initRepo(repo types.Repo, dest types.Radicle, tempdir string, env []string) (string, error) {
	branch, err := run(tempdir, env, "git", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", err
	}

	description := strings.TrimSpace(fmt.Sprintf("%s %s", repo.Description, marker(repo.URL)))

	visibility, err := visibilityFlag(dest, repo)
	if err != nil {
		return "", err
	}

	_, err = run(tempdir, env, "rad", "init",
		fmt.Sprintf("--name=%s", repo.Name),
		fmt.Sprintf("--description=%s", description),
		fmt.Sprintf("--default-branch=%s", branch),
		visibility,
		"--no-confirm")
	if err != nil {
		return "", err
	}

	rid, err := run(tempdir, env, "rad", ".")
	if err != nil {
		return "", err
	}

	rid = strings.TrimPrefix(rid, "rad:")

	source, err := json.Marshal(stripCredentials(repo.URL))
	if err != nil {
		return rid, err
	}

	if err := recordMirrorSource(tempdir, env, source, rid); err != nil {
		return rid, err
	}

	updatePayloadCache(repo, rid)

	return rid, nil
}

// updatePayloadCache Remember the new mirror so a later run in this process finds it without
// re-inspecting storage — and never tries to initialize it a second time.
func updatePayloadCache(repo types.Repo, rid string) {
	home, err := Home()
	if err != nil {
		return
	}
	if payloadCache[home] == nil {
		payloadCache[home] = map[string]payload{}
	}
	var newPayload payload
	newPayload.Gickup.Source = stripCredentials(repo.URL)
	payloadCache[home][rid] = newPayload
}

// recordMirrorSource Captures the upstream source, so an existing mirror can be reused,
// avoiding duplicates. A failure here fails the overall
// backup, and the error names the repository because the storage offers
// no rollback for a created identity: the operator has to record the
// source or remove the repository by hand.
func recordMirrorSource(tempdir string, env []string, source []byte, rid string) error {
	if _, err := run(tempdir, env, "rad", "id", "update",
		"--payload", "dev.gickup", "source", string(source),
		"--title", "Record mirror source",
		"--no-confirm"); err != nil {
		return fmt.Errorf(
			"repository rad:%s was created but recording its mirror source failed - record dev.gickup.source manually or remove the repository: %w",
			rid, err)
	}
	return nil
}

// visibilityFlag maps the configured visibility to the rad init flag.
func visibilityFlag(dest types.Radicle, repo types.Repo) (string, error) {
	switch dest.Visibility {
	case "public":
		return "--public", nil
	case "private":
		return "--private", nil
	case "", "source":
		if repo.Private {
			return "--private", nil
		}

		return "--public", nil
	default:
		return "", fmt.Errorf("invalid radicle visibility %s, must be public, private or source", dest.Visibility)
	}
}

// pushArgs builds the git arguments that mirror all branches and tags into
// the profile's namespace of the stored repository.
func pushArgs(dest types.Radicle, rid, nid string) []string {
	prefix := ""
	if dest.Force {
		prefix = "+"
	}

	args := []string{"push"}
	if dest.Prune {
		args = append(args, "--prune")
	}

	return append(args,
		fmt.Sprintf("rad://%s/%s", rid, nid),
		fmt.Sprintf("%srefs/heads/*:refs/heads/*", prefix),
		fmt.Sprintf("%srefs/tags/*:refs/tags/*", prefix))
}

// push mirrors all branches and tags of the temporary clone into the profile's
// namespace of the stored repository. The remote helper updates and signs
// rad/sigrefs and announces to the node, so nothing else has to touch storage.
func push(dest types.Radicle, tempdir, rid, nid string, env []string) error {
	_, err := run(tempdir, env, "git", pushArgs(dest, rid, nid)...)

	return err
}

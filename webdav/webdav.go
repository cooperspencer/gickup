package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
)

var sub zerolog.Logger

func joinURL(base string, segments ...string) string {
	joined := strings.TrimRight(base, "/")
	for _, seg := range segments {
		seg = strings.Trim(seg, "/")
		if seg == "" {
			continue
		}
		joined += "/" + seg
	}
	return joined
}

func request(client *http.Client, method, urlStr, user, pass string, contentLength int64, body io.ReadSeeker) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	return doWithRetry(client, req, body)
}

// newClient returns an *http.Client based on the default transport with a
// ResponseHeaderTimeout so a stalled server cannot leak goroutines; no overall
// Timeout is set, so large uploads are not aborted mid-transfer.
func newClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: t}
}

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	resp.Body.Close()
}

const (
	retryAttempts = 3
	retryBaseWait = 500 * time.Millisecond
)

// shouldRetry reports whether a transport error or response status warrants a
// retry. All methods issued by this package (MKCOL, PUT, DELETE, PROPFIND) are
// idempotent or read-only, so retrying them is safe.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// doWithRetry runs client.Do with bounded exponential backoff for transient
// failures. body, when non-nil, is rewound before each attempt so PUTs repeat
// the full payload instead of a truncated or empty body.
func doWithRetry(client *http.Client, req *http.Request, body io.ReadSeeker) (*http.Response, error) {
	var (
		resp    *http.Response
		err     error
		backoff = retryBaseWait
	)
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		if body != nil {
			if attempt > 1 {
				if _, serr := body.Seek(0, io.SeekStart); serr != nil {
					return nil, fmt.Errorf("retry %s %s: rewind body: %w", req.Method, req.URL.Redacted(), serr)
				}
			}
			req.Body = io.NopCloser(body)
		}
		resp, err = client.Do(req)
		if !shouldRetry(resp, err) {
			return resp, err
		}
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		if attempt == retryAttempts {
			break
		}
		wait := backoff + time.Duration(rand.Int64N(int64(backoff/2)))
		sub.Warn().Msgf("%s %s: attempt %d failed (%s); retrying in %v", req.Method, req.URL.Redacted(), attempt, retryReason(resp, err), wait)
		time.Sleep(wait)
		backoff *= 2
	}
	return resp, err
}

func retryReason(resp *http.Response, err error) string {
	if err != nil {
		return err.Error()
	}
	return resp.Status
}

// ensureCollections MKCOLs every collection along the repo path and the
// file's directory segments, skipping ones already created in this run.
func ensureCollections(client *http.Client, repo types.WebDAVRepo, segments []string, created map[string]bool) error {
	current := strings.TrimRight(repo.Url, "/")
	parts := append(strings.Split(repo.Path, "/"), segments...)
	for _, seg := range parts {
		seg = strings.Trim(seg, "/")
		if seg == "" {
			continue
		}
		current += "/" + seg
		if created[current] {
			continue
		}
		resp, err := request(client, "MKCOL", current, repo.Username, repo.Password, -1, nil)
		if err != nil {
			return err
		}
		// 201 Created, 405 Method Not Allowed (collection already exists) or 200 OK.
		if resp.StatusCode != http.StatusCreated &&
			resp.StatusCode != http.StatusMethodNotAllowed &&
			resp.StatusCode != http.StatusOK {
			return fmt.Errorf("MKCOL %s: %s", current, resp.Status)
		}
		drainAndClose(resp)
		created[current] = true
	}
	return nil
}

// UploadDirToWebDAV uploads the contents of a directory to a WebDAV server via HTTP PUT with Basic auth.
func UploadDirToWebDAV(directory string, repo types.WebDAVRepo) error {
	sub = logger.CreateSubLogger("stage", "webdav", "url", repo.Url)
	client := newClient()
	created := map[string]bool{}

	err := filepath.Walk(directory, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		rel := filepath.ToSlash(p[len(directory)+1:])
		relDir, _ := path.Split(rel)
		dirSegments := strings.Split(strings.Trim(relDir, "/"), "/")
		if err := ensureCollections(client, repo, dirSegments, created); err != nil {
			return err
		}

		file, err := os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		remoteURL := joinURL(repo.Url, repo.Path, rel)
		resp, err := request(client, http.MethodPut, remoteURL, repo.Username, repo.Password, info.Size(), file)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			return fmt.Errorf("PUT %s: %s %s", remoteURL, resp.Status, strings.TrimSpace(string(body)))
		}
		drainAndClose(resp)
		return nil
	})
	return err
}

// davResponse is a single response element in a WebDAV multistatus body.
type davResponse struct {
	Href     string `xml:"href"`
	Propstat []struct {
		Prop struct {
			Resourcetype []byte `xml:",innerxml"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

func (r davResponse) isCollection() bool {
	for _, ps := range r.Propstat {
		if bytes.Contains(ps.Prop.Resourcetype, []byte("collection")) {
			return true
		}
	}
	return false
}

// errDepthInfinityRejected signals that the server refused the Depth: infinity
// PROPFIND, so the caller should fall back to a recursive Depth: 1 traversal.
var errDepthInfinityRejected = errors.New("webdav server rejected Depth: infinity PROPFIND")

// propfind lists all resources under rootURL. It prefers a single Depth:
// infinity PROPFIND; servers that refuse infinity (Sabre/DAV, Nextcloud and
// others answer 400/403/501) get an emulating recursive Depth: 1 walk. A 404
// (collection does not exist yet) yields no error.
func propfind(client *http.Client, rootURL string, repo types.WebDAVRepo) ([]davResponse, error) {
	resources, err := propfindDepth(client, rootURL, "infinity", repo)
	if err == errDepthInfinityRejected {
		sub.Info().Msg("webdav server does not support Depth: infinity; falling back to recursive PROPFIND")
		return propfindRecursive(client, rootURL, repo)
	}
	return resources, err
}

// propfindDepth issues a single PROPFIND with the given Depth header and parses
// the multistatus body. A 404 yields no error and no resources.
func propfindDepth(client *http.Client, rootURL, depth string, repo types.WebDAVRepo) ([]davResponse, error) {
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<D:propfind xmlns:D="DAV:"><D:prop><D:resourcetype/></D:prop></D:propfind>`
	req, err := http.NewRequestWithContext(context.Background(), "PROPFIND", rootURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(repo.Username, repo.Password)
	req.Header.Set("Depth", depth)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := doWithRetry(client, req, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if depth == "infinity" && depthInfinityRejected(resp.StatusCode, snippet) {
			return nil, errDepthInfinityRejected
		}
		return nil, fmt.Errorf("PROPFIND %s: %s %s", rootURL, resp.Status, strings.TrimSpace(string(snippet)))
	}

	return parseMultistatus(resp.Body)
}

// depthInfinityRejected reports whether a non-success PROPFIND response means
// the server refuses Depth: infinity. 501 is the canonical "feature not
// implemented" code; for 400/403 the body must also mention depth/infinity so a
// genuine permission or malformed-request error still surfaces instead of
// triggering a silent fallback. doWithRetry already absorbed transient 5xx, so
// a response reaching here is deterministic.
func depthInfinityRejected(status int, bodySnippet []byte) bool {
	switch status {
	case http.StatusNotImplemented:
		return true
	case http.StatusBadRequest, http.StatusForbidden:
		s := strings.ToLower(string(bodySnippet))
		return strings.Contains(s, "infinity") || strings.Contains(s, "depth")
	}
	return false
}

// propfindRecursive emulates a Depth: infinity PROPFIND by issuing Depth: 1
// PROPFINDs to rootURL and every nested collection, deduplicating resources by
// their repo-relative key so each collection's self-entry is reported once and
// symlink-like cycles cannot loop forever.
func propfindRecursive(client *http.Client, rootURL string, repo types.WebDAVRepo) ([]davResponse, error) {
	bpURL, err := url.Parse(joinURL(repo.Url, repo.Path))
	if err != nil {
		return nil, err
	}
	collected := make(map[string]davResponse)
	var order []string
	visited := make(map[string]bool)

	var walk func(collectionURL string) error
	walk = func(collectionURL string) error {
		ck := relKey(collectionURL, bpURL.Path)
		if visited[ck] {
			return nil
		}
		visited[ck] = true
		resources, err := propfindDepth(client, collectionURL, "1", repo)
		if err != nil {
			return err
		}
		for _, r := range resources {
			hk := relKey(r.Href, bpURL.Path)
			if _, ok := collected[hk]; !ok {
				collected[hk] = r
				order = append(order, hk)
			}
			if r.isCollection() && hk != ck && !visited[hk] {
				if err := walk(absoluteURL(r.Href, repo.Url)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(rootURL); err != nil {
		return nil, err
	}

	out := make([]davResponse, 0, len(order))
	for _, k := range order {
		out = append(out, collected[k])
	}
	return out, nil
}

// relKey extracts the object key relative to basePath from a WebDAV href, which may be an absolute path, a full URL or percent-encoded.
func relKey(href, basePath string) string {
	h := href
	if strings.Contains(h, "://") {
		rest := h[strings.Index(h, "://")+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			h = rest[slash:]
		} else {
			h = "/"
		}
	}
	if unescaped, err := url.PathUnescape(h); err == nil {
		h = unescaped
	}
	prefix := strings.Trim(basePath, "/")
	if prefix != "" {
		h = strings.TrimPrefix(h, "/"+prefix+"/")
	} else {
		h = strings.TrimPrefix(h, "/")
	}
	return strings.Trim(h, "/")
}

func parseMultistatus(r io.Reader) ([]davResponse, error) {
	var ms struct {
		XMLName   xml.Name      `xml:"multistatus"`
		Responses []davResponse `xml:"response"`
	}
	if err := xml.NewDecoder(r).Decode(&ms); err != nil {
		return nil, fmt.Errorf("parse PROPFIND response: %w", err)
	}
	return ms.Responses, nil
}

// absoluteURL turns a possibly-relative WebDAV href into an absolute URL.
func absoluteURL(href, baseURL string) string {
	if strings.Contains(href, "://") {
		return href
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	return u.Scheme + "://" + u.Host + href
}

// DeleteObjectsNotInRepo deletes files from the WebDAV server that are not present in the repository.
func DeleteObjectsNotInRepo(directory, repoName string, repo types.WebDAVRepo) error {
	sub = logger.CreateSubLogger("stage", "webdav", "url", repo.Url)
	client := newClient()

	basePath := joinURL(repo.Url, repo.Path)
	rootURL := joinURL(basePath, repoName)

	resources, err := propfind(client, rootURL, repo)
	if err != nil {
		return err
	}

	basePathURL, err := url.Parse(basePath)
	if err != nil {
		return err
	}

	for _, res := range resources {
		if res.isCollection() {
			continue
		}
		key := relKey(res.Href, basePathURL.Path)
		if key == "" {
			continue
		}
		localPath := filepath.Join(directory, filepath.FromSlash(key))
		rel, err := filepath.Rel(directory, localPath)
		if err != nil || strings.HasPrefix(filepath.ToSlash(rel), "..") {
			sub.Warn().Msgf("skipping webdav path outside backup tree: %s", key)
			continue
		}
		if _, err := os.Stat(localPath); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			sub.Debug().Msgf("Removing %s from webdav", key)
			resp, err := request(client, http.MethodDelete, absoluteURL(res.Href, repo.Url), repo.Username, repo.Password, -1, nil)
			if err != nil {
				return err
			}
			if resp.StatusCode < 200 || (resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound) {
				status := resp.Status
				drainAndClose(resp)
				return fmt.Errorf("DELETE %s: %s", res.Href, status)
			}
			drainAndClose(resp)
		}
	}

	return nil
}

package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func request(client *http.Client, method, urlStr, user, pass string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	return client.Do(req)
}

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	resp.Body.Close()
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
		resp, err := request(client, "MKCOL", current, repo.Username, repo.Password, nil)
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
	client := &http.Client{}
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
		resp, err := request(client, http.MethodPut, remoteURL, repo.Username, repo.Password, file)
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

// propfind lists all resources under rootURL via a Depth: infinity PROPFIND; a 404 (collection does not exist yet) yields no error.
func propfind(client *http.Client, rootURL string, repo types.WebDAVRepo) ([]davResponse, error) {
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<D:propfind xmlns:D="DAV:"><D:prop><D:resourcetype/></D:prop></D:propfind>`
	req, err := http.NewRequestWithContext(context.Background(), "PROPFIND", rootURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(repo.Username, repo.Password)
	req.Header.Set("Depth", "infinity")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("PROPFIND %s: %s %s", rootURL, resp.Status, strings.TrimSpace(string(b)))
	}

	return parseMultistatus(resp.Body)
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
	client := &http.Client{}

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
		if _, err := os.Stat(localPath); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			sub.Debug().Msgf("Removing %s from webdav", key)
			resp, err := request(client, http.MethodDelete, absoluteURL(res.Href, repo.Url), repo.Username, repo.Password, nil)
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

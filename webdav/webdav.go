package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
	gowebdav "github.com/studio-b12/gowebdav"
)

var sub zerolog.Logger

func newClient(repo types.WebDAVRepo) *gowebdav.Client {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t = &http.Transport{}
	}
	t = t.Clone()
	t.ResponseHeaderTimeout = 60 * time.Second
	c := gowebdav.NewClient(repo.Url, repo.Username, repo.Password)
	c.SetTransport(&retryRoundTripper{base: t})
	return c
}

// UploadDirToWebDAV uploads the contents of a directory to a WebDAV server
func UploadDirToWebDAV(directory string, repo types.WebDAVRepo) error {
	sub = logger.CreateSubLogger("stage", "webdav", "url", repo.Url)
	c := newClient(repo)

	return filepath.Walk(directory, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel := filepath.ToSlash(p[len(directory)+1:])
		remote := path.Join(repo.Path, rel)

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		return c.WriteStreamWithLength(remote, f, info.Size(), 0o644)
	})
}

// DeleteObjectsNotInRepo deletes files from the WebDAV server that are not present in the repository
func DeleteObjectsNotInRepo(directory, repoName string, repo types.WebDAVRepo) error {
	sub = logger.CreateSubLogger("stage", "webdav", "url", repo.Url)
	c := newClient(repo)
	root := path.Join(repo.Path, repoName)

	keys, err := walkRemote(c, root)
	if err != nil {
		// Nothing uploaded yet.
		if gowebdav.IsErrNotFound(err) {
			return nil
		}
		return err
	}
	for _, key := range keys {
		if _, statErr := os.Stat(filepath.Join(directory, repoName, filepath.FromSlash(key))); statErr == nil {
			continue
		} else if !os.IsNotExist(statErr) {
			return statErr
		}
		sub.Debug().Msgf("Removing %s from webdav", key)
		if err := c.RemoveAll(path.Join(root, key)); err != nil {
			return err
		}
	}

	return nil
}

// walkRemote returns the keys relative to dir of every regular file beneath it.
func walkRemote(c *gowebdav.Client, dir string) ([]string, error) {
	var files []string
	prefix := strings.TrimSuffix(dir, "/") + "/"
	visited := map[string]bool{}

	var walk func(current string) error
	walk = func(current string) error {
		if visited[current] {
			return nil
		}
		visited[current] = true
		entries, err := c.ReadDir(current)
		if err != nil {
			return err
		}
		for _, e := range entries {
			child := path.Join(current, e.Name())
			if e.IsDir() {
				if err := walk(child); err != nil {
					return err
				}
				continue
			}
			files = append(files, strings.TrimPrefix(child, prefix))
		}
		return nil
	}
	if err := walk(dir); err != nil {
		return nil, err
	}
	return files, nil
}

// retryRoundTripper retries transient failures with exponential backoff,
// rewinding seekable bodies via GetBody so PUTs replay the full payload.
type retryRoundTripper struct {
	base http.RoundTripper
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	const maxAttempts = 3
	backoff := 500 * time.Millisecond

	resp, err := r.base.RoundTrip(req)
	for attempt := 1; shouldRetry(resp, err) && attempt < maxAttempts; attempt++ {
		// Drain and close the failed response so its connection can be reused.
		if resp != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
			resp.Body.Close()
		}

		// Rewind seekable bodies via GetBody so retried PUTs replay the full payload.
		if req.GetBody != nil {
			body, berr := req.GetBody()
			if berr != nil {
				return nil, fmt.Errorf("retry %s %s: rewind body: %w", req.Method, req.URL.Redacted(), berr)
			}
			req.Body = body
		}

		wait := backoff + time.Duration(rand.Int64N(int64(backoff/2)))
		reason := "unknown"
		switch {
		case err != nil:
			reason = err.Error()
		case resp != nil:
			reason = resp.Status
		}
		sub.Warn().Msgf("%s %s: attempt %d failed (%s); retrying in %v",
			req.Method, req.URL.Redacted(), attempt, reason, wait)

		// Back off with jitter, bailing out immediately if the request was cancelled.
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			timer.Stop()
			return resp, req.Context().Err()
		}
		backoff *= 2

		resp, err = r.base.RoundTrip(req)
	}
	return resp, err
}

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// Context cancellations are intentional, not transient
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		// Only retry transient network failures
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return true
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return true
		}
		if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
			errors.Is(err, syscall.EPIPE) {
			return true
		}
		return false
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

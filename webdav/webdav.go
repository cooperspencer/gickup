package webdav

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/types"
)

const (
	DefaultConnectTimeout = 30 * time.Second
	DefaultRequestTimeout = 5 * time.Minute
	MaxRetries            = 3
	BaseBackoff           = 1 * time.Second
)

type WebDAVClient interface {
	Mkcol(ctx context.Context, path string) error
	Put(ctx context.Context, path string, body io.Reader, contentType string) error
	Propfind(ctx context.Context, path string) (*DavResponse, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	Close() error
}

type DavResponse struct {
	StatusCode int
	Properties map[string]string
	Responses  []struct {
		Href     string `xml:"href"`
		PropStat []struct {
			Prop struct {
				GetLastModified  string `xml:"getlastmodified"`
				GetContentLength int64  `xml:"getcontentlength"`
			} `xml:"prop"`
			Status string `xml:"status"`
		} `xml:"propstat"`
	} `xml:"response"`
}

func (r *DavResponse) ParseXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	return decoder.Decode(r)
}

type webdavClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewWebDAVClient(url, username, password string) WebDAVClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	return &webdavClient{
		baseURL:  strings.TrimSuffix(url, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   DefaultRequestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return &WebDAVError{Message: "too many redirects"}
				}
				return nil
			},
		},
	}
}

func (c *webdavClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *webdavClient) doRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if c.username != "" && c.password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *webdavClient) withRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	backoff := BaseBackoff

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if err := operation(); err != nil {
			lastErr = err
			if isTransientError(err) && attempt < MaxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return err
		}
		return nil
	}

	return lastErr
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	if davErr, ok := err.(*WebDAVError); ok {
		return davErr.StatusCode >= 500 && davErr.StatusCode < 600
	}

	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "i/o timeout")
}

func (c *webdavClient) Mkcol(ctx context.Context, path string) error {
	return c.withRetry(ctx, func() error {
		resp, err := c.doRequest(ctx, "MKCOL", path, nil, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMethodNotAllowed {
			return &WebDAVError{StatusCode: resp.StatusCode, Message: "MKCOL failed"}
		}
		return nil
	})
}

func (c *webdavClient) Put(ctx context.Context, path string, body io.Reader, contentType string) error {
	headers := map[string]string{
		"Content-Type": contentType,
	}
	return c.withRetry(ctx, func() error {
		resp, err := c.doRequest(ctx, "PUT", path, body, headers)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			return &WebDAVError{StatusCode: resp.StatusCode, Message: "PUT failed"}
		}
		return nil
	})
}

func (c *webdavClient) Propfind(ctx context.Context, path string) (*DavResponse, error) {
	headers := map[string]string{
		"Depth":        "1",
		"Content-Type": "application/xml",
	}
	body := `<?xml version="1.0" encoding="utf-8"?><D:propfind xmlns:D="DAV:"><D:prop></D:prop></D:propfind>`

	resp, err := c.doRequest(ctx, "PROPFIND", path, strings.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusMultiStatus {
		return nil, &WebDAVError{StatusCode: resp.StatusCode, Message: "PROPFIND failed"}
	}

	davResp := &DavResponse{StatusCode: resp.StatusCode}
	if err := davResp.ParseXML(resp.Body); err != nil {
		return nil, err
	}

	return davResp, nil
}

func (c *webdavClient) Delete(ctx context.Context, path string) error {
	return c.withRetry(ctx, func() error {
		resp, err := c.doRequest(ctx, "DELETE", path, nil, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
			return &WebDAVError{StatusCode: resp.StatusCode, Message: "DELETE failed"}
		}
		return nil
	})
}

func (c *webdavClient) Exists(ctx context.Context, path string) (bool, error) {
	resp, err := c.doRequest(ctx, "HEAD", path, nil, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

type WebDAVError struct {
	StatusCode int
	Message    string
}

func (e *WebDAVError) Error() string {
	return e.Message
}

func NormalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "/")
	return path
}

func CreateParentDirectories(ctx context.Context, client WebDAVClient, path string) error {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	currentPath := ""

	for i := 0; i < len(parts)-1; i++ {
		if currentPath == "" {
			currentPath = "/" + parts[i]
		} else {
			currentPath = currentPath + "/" + parts[i]
		}
		if err := client.Mkcol(ctx, currentPath); err != nil {
			if _, ok := err.(*WebDAVError); !ok {
				return err
			}
		}
	}

	return nil
}

func UploadWithRetry(ctx context.Context, client WebDAVClient, path string, body io.Reader, contentType string) error {
	return client.Put(ctx, path, body, contentType)
}

func PropfindWithRetry(ctx context.Context, client WebDAVClient, path string) (*DavResponse, error) {
	return client.Propfind(ctx, path)
}

func IsEmptyResponse(resp *DavResponse) bool {
	return resp == nil || len(resp.Responses) == 0
}

func DavResponseToPaths(resp *DavResponse) []string {
	if resp == nil {
		return nil
	}
	paths := make([]string, 0, len(resp.Responses))
	for _, r := range resp.Responses {
		for _, ps := range r.PropStat {
			if strings.Contains(ps.Status, "200 OK") {
				paths = append(paths, r.Href)
				break
			}
		}
	}
	return paths
}

func DeleteWithRetry(ctx context.Context, client WebDAVClient, path string) error {
	return client.Delete(ctx, path)
}

func UploadDirToWebDAV(ctx context.Context, directory string, repo types.Repo, config types.WebDAV, dryRun bool) error {
	if dryRun {
		return nil
	}

	client := NewWebDAVClient(config.URL, config.Username, config.Password)
	defer client.Close()

	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || (info.Mode()&os.ModeSymlink != 0) {
			return nil
		}

		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		webdavPath := NormalizePath(relPath)
		if config.Path == "" && config.Structured {
			webdavPath = repo.Hoster + "/" + repo.Owner + "/" + webdavPath
		}
		if config.Path != "" {
			webdavPath = NormalizePath(config.Path) + "/" + webdavPath
		}

		fullPath := "/" + webdavPath

		err = CreateParentDirectories(ctx, client, fullPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		return UploadWithRetry(ctx, client, fullPath, file, "application/octet-stream")
	})
}

func DeleteObjectsNotInRepo(ctx context.Context, directory string, repo types.Repo, config types.WebDAV, dryRun bool) error {
	if dryRun {
		return nil
	}

	client := NewWebDAVClient(config.URL, config.Username, config.Password)
	defer client.Close()

	remotePath := NormalizePath(config.Path)
	if remotePath != "" {
		remotePath = "/" + remotePath
	}

	resp, err := PropfindWithRetry(ctx, client, remotePath)
	if err != nil {
		return err
	}

	if resp == nil || IsEmptyResponse(resp) {
		return nil
	}

	remotePaths := DavResponseToPaths(resp)

	localFiles := make(map[string]bool)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || (info.Mode()&os.ModeSymlink != 0) {
			return nil
		}
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return nil
		}
		localFiles[relPath] = true
		return nil
	})

	for _, remoteFile := range remotePaths {
		relRemotePath := strings.TrimPrefix(remoteFile, remotePath+"/")
		if relRemotePath == "" {
			continue
		}
		if !localFiles[relRemotePath] {
			err = DeleteWithRetry(ctx, client, remoteFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

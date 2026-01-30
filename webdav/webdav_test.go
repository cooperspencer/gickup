package webdav

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cooperspencer/gickup/types"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "README.md",
			expected: "README.md",
		},
		{
			name:     "nested path",
			input:    "subdir/file.txt",
			expected: "subdir/file.txt",
		},
		{
			name:     "windows separators",
			input:    "subdir\\file.txt",
			expected: "subdir/file.txt",
		},
		{
			name:     "leading slash removed",
			input:    "/backups/repo",
			expected: "backups/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDavResponseToPaths(t *testing.T) {
	resp := &DavResponse{
		Responses: []struct {
			Href     string `xml:"href"`
			PropStat []struct {
				Prop struct {
					GetLastModified  string `xml:"getlastmodified"`
					GetContentLength int64  `xml:"getcontentlength"`
				} `xml:"prop"`
				Status string `xml:"status"`
			} `xml:"propstat"`
		}{
			{
				Href: "/webdav/file1.txt",
				PropStat: []struct {
					Prop struct {
						GetLastModified  string `xml:"getlastmodified"`
						GetContentLength int64  `xml:"getcontentlength"`
					} `xml:"prop"`
					Status string `xml:"status"`
				}{
					{
						Status: "HTTP/1.1 200 OK",
					},
				},
			},
			{
				Href: "/webdav/file2.txt",
				PropStat: []struct {
					Prop struct {
						GetLastModified  string `xml:"getlastmodified"`
						GetContentLength int64  `xml:"getcontentlength"`
					} `xml:"prop"`
					Status string `xml:"status"`
				}{
					{
						Status: "HTTP/1.1 404 Not Found",
					},
				},
			},
		},
	}

	paths := DavResponseToPaths(resp)
	if len(paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(paths))
	}

	if paths[0] != "/webdav/file1.txt" {
		t.Errorf("Expected /webdav/file1.txt, got %s", paths[0])
	}
}

func TestIsEmptyResponse(t *testing.T) {
	if !IsEmptyResponse(&DavResponse{}) {
		t.Error("Expected empty response to be true")
	}

	if IsEmptyResponse(&DavResponse{Responses: []struct {
		Href     string `xml:"href"`
		PropStat []struct {
			Prop struct {
				GetLastModified  string `xml:"getlastmodified"`
				GetContentLength int64  `xml:"getcontentlength"`
			} `xml:"prop"`
			Status string `xml:"status"`
		} `xml:"propstat"`
	}{{}}}) {
		t.Error("Expected non-empty response to be false")
	}
}

func TestRepoStructure(t *testing.T) {
	repo := types.Repo{
		Name:   "owner/repo",
		URL:    "https://github.com/owner/repo",
		Owner:  "owner",
		Hoster: "github.com",
	}

	if repo.Name != "owner/repo" {
		t.Errorf("Expected repo name 'owner/repo', got '%s'", repo.Name)
	}

	if repo.Hoster != "github.com" {
		t.Errorf("Expected hoster 'github.com', got '%s'", repo.Hoster)
	}
}

func TestWebDAVConfigStructure(t *testing.T) {
	config := types.WebDAV{
		URL:           "https://example.com/webdav",
		Username:      "user",
		Password:      "pass",
		Path:          "/backups",
		Structured:    true,
		Zip:           true,
		DateCreateDir: true,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	normalizedPath := config.GetNormalizedPath()
	if normalizedPath != "backups" {
		t.Errorf("Expected normalized path 'backups', got '%s'", normalizedPath)
	}
}

func TestHTTPClientConfiguration(t *testing.T) {
	client := NewWebDAVClient("https://example.com/webdav", "user", "pass")
	wdClient := client.(*webdavClient)

	if wdClient.httpClient == nil {
		t.Error("httpClient should not be nil")
	}

	if wdClient.httpClient.Timeout != DefaultRequestTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultRequestTimeout, wdClient.httpClient.Timeout)
	}

	transport, ok := wdClient.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected http.Client to use *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Error("TLSClientConfig should be configured")
	}

	if transport.MaxIdleConns == 0 {
		t.Error("MaxIdleConns should be set for connection reuse")
	}
}

func TestBasicAuthHeaderGeneration(t *testing.T) {
	client := NewWebDAVClient("https://example.com/webdav", "testuser", "testpass")
	wdClient := client.(*webdavClient)

	expected := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	received := base64.StdEncoding.EncodeToString([]byte(wdClient.username + ":" + wdClient.password))

	if received != expected {
		t.Errorf("Expected auth header value %s, got %s", expected, received)
	}
}

func TestRetryConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		maxRetries  int
		baseBackoff time.Duration
		expectCalls int
	}{
		{
			name:        "default retry config",
			maxRetries:  MaxRetries,
			baseBackoff: BaseBackoff,
			expectCalls: MaxRetries + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.maxRetries != 3 {
				t.Errorf("Expected maxRetries 3, got %d", tt.maxRetries)
			}
			if tt.baseBackoff != 1*time.Second {
				t.Errorf("Expected baseBackoff 1s, got %v", tt.baseBackoff)
			}
		})
	}
}

func TestTransientErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "500 error is transient",
			err:      &WebDAVError{StatusCode: 500},
			expected: true,
		},
		{
			name:     "502 error is transient",
			err:      &WebDAVError{StatusCode: 502},
			expected: true,
		},
		{
			name:     "503 error is transient",
			err:      &WebDAVError{StatusCode: 503},
			expected: true,
		},
		{
			name:     "401 error is not transient",
			err:      &WebDAVError{StatusCode: 401},
			expected: false,
		},
		{
			name:     "403 error is not transient",
			err:      &WebDAVError{StatusCode: 403},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("isTransientError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestWebDAVErrorImplementsError(t *testing.T) {
	var err error = &WebDAVError{StatusCode: 500, Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", err.Error())
	}
}

func TestDavResponseXMLParsing(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/path/to/resource</d:href>
    <d:propstat>
      <d:prop>
        <d:getlastmodified>Sun, 01 Jan 2023 00:00:00 GMT</d:getlastmodified>
        <d:getcontentlength>12345</d:getcontentlength>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

	resp := &DavResponse{StatusCode: 207}
	err := resp.ParseXML(strings.NewReader(xmlData))
	if err != nil {
		t.Errorf("Failed to parse XML: %v", err)
	}

	if len(resp.Responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(resp.Responses))
	}

	if resp.Responses[0].Href != "/path/to/resource" {
		t.Errorf("Expected href '/path/to/resource', got '%s'", resp.Responses[0].Href)
	}
}

func TestDavResponseUnmarshalXML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
  <D:response>
    <D:href>/webdav/file.txt</D:href>
    <D:propstat>
      <D:prop>
        <D:getlastmodified>Mon, 02 Jan 2023 12:00:00 GMT</D:getlastmodified>
        <D:getcontentlength>500</D:getcontentlength>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
</D:multistatus>`

	resp := &DavResponse{}
	err := xml.Unmarshal([]byte(xmlData), resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if len(resp.Responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(resp.Responses))
	}
}

func TestUploadDirToWebDAV(t *testing.T) {
	var uploadedFiles []string
	var createdDirs []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "MKCOL" {
			createdDirs = append(createdDirs, r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		} else if r.Method == "PUT" {
			uploadedFiles = append(uploadedFiles, r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "webdav-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := types.WebDAV{URL: server.URL}

	repo := types.Repo{Name: "test-repo"}
	err = UploadDirToWebDAV(context.Background(), tmpDir, repo, config, false)
	if err != nil {
		t.Errorf("UploadDirToWebDAV() error = %v", err)
	}

	if len(uploadedFiles) == 0 {
		t.Error("Expected files to be uploaded")
	}
}

func TestDeleteObjectsNotInRepo(t *testing.T) {
	var deletedFiles []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deletedFiles = append(deletedFiles, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		} else if r.Method == "PROPFIND" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusMultiStatus)
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/repo/old-file.txt</d:href>
    <d:propstat>
      <d:prop>
        <d:getcontentlength>100</d:getcontentlength>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "webdav-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	newFile := filepath.Join(tmpDir, "new-file.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to write new file: %v", err)
	}

	repo := types.Repo{Name: "repo"}
	config := types.WebDAV{URL: server.URL}

	err = DeleteObjectsNotInRepo(context.Background(), tmpDir, repo, config, false)
	if err != nil {
		t.Errorf("DeleteObjectsNotInRepo() error = %v", err)
	}

	found := false
	for _, f := range deletedFiles {
		if strings.Contains(f, "old-file.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected old-file.txt to be deleted")
	}
}

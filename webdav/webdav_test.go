package webdav

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRelKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		href, basePath, want string
	}{
		{"/dav/repos/r/objects/ab/c", "/dav/repos", "r/objects/ab/c"},
		{"https://h/dav/repos/r/HEAD", "/dav/repos", "r/HEAD"},
		{"/dav/repos/r%20name/f", "/dav/repos", "r name/f"},
		{"/dav/repos/r/", "/dav/repos", "r"},
	}
	for _, c := range cases {
		if got := relKey(c.href, c.basePath); got != c.want {
			t.Errorf("relKey(%q, %q) = %q, want %q", c.href, c.basePath, got, c.want)
		}
	}
}

func TestParseMultistatus(t *testing.T) {
	t.Parallel()

	body := `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
  <D:response><D:href>/dav/repos/r/HEAD</D:href>
    <D:propstat><D:prop><D:resourcetype/></D:prop></D:propstat></D:response>
  <D:response><D:href>/dav/repos/r/</D:href>
    <D:propstat><D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop></D:propstat></D:response>
</D:multistatus>`

	res, err := parseMultistatus(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parseMultistatus: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d responses, want 2", len(res))
	}
	if res[0].isCollection() {
		t.Error("HEAD must not be a collection")
	}
	if !res[1].isCollection() {
		t.Error("r/ must be a collection")
	}
}

// TestRequestRetriesOnTransientFailures asserts that doWithRetry retries 5xx
// responses, rewinds the PUT body on each attempt, and surfaces the final
// success status. It is the one runnable check for the retry loop added in
// this package; backoff sleeps (~1.5s) are accepted to keep production code
// free of test-only knobs.
func TestRequestRetriesOnTransientFailures(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 1024)
	var (
		hits    atomic.Int32
		lengths []int
		mu      sync.Mutex
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		lengths = append(lengths, len(b))
		mu.Unlock()
		if hits.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := newClient()
	resp, err := request(client, http.MethodPut, srv.URL, "u", "p", int64(len(payload)), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer drainAndClose(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 after retries", resp.StatusCode)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("server hits = %d, want 3", got)
	}
	mu.Lock()
	defer mu.Unlock()
	for i, n := range lengths {
		if n != len(payload) {
			t.Fatalf("attempt %d body length = %d, want %d (body rewind broken)", i+1, n, len(payload))
		}
	}
}

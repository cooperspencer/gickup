package httpClient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgentTransportRewrites(t *testing.T) {
	t.Parallel()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	rt := userAgentTransport{base: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "go-git/5.x")
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	resp.Body.Close()

	if got != gickupUserAgent {
		t.Fatalf("expected User-Agent %q, got %q", gickupUserAgent, got)
	}
}

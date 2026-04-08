package apprise

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestNotifySendsExpectedRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notify/team" {
			t.Fatalf("path = %q, want /notify/team", r.URL.Path)
		}

		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q, want application/json", got)
		}

		payload := Request{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if payload.Body != "backup complete" {
			t.Fatalf("body = %q, want backup complete", payload.Body)
		}

		if len(payload.Tags) != 2 || payload.Tags[0] != "backup" || payload.Tags[1] != "nightly" {
			t.Fatalf("unexpected tags: %#v", payload.Tags)
		}

		if len(payload.Urls) != 1 || payload.Urls[0] != "discord://room" {
			t.Fatalf("unexpected urls: %#v", payload.Urls)
		}

		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	err := Notify("backup complete", types.AppriseConfig{
		Url:    server.URL,
		Config: "team",
		Tags:   []string{"backup", "nightly"},
		Urls:   []string{"discord://room"},
	})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestNotifyReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"push failed"}`))
	}))
	defer server.Close()

	err := Notify("backup complete", types.AppriseConfig{Url: server.URL})
	if err == nil {
		t.Fatal("expected apprise error")
	}
}

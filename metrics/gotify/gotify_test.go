package gotify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestNotifySendsExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message" {
			t.Fatalf("path = %q, want /message", r.URL.Path)
		}

		if got := r.URL.Query().Get("token"); got != "secret-token" {
			t.Fatalf("token = %q, want secret-token", got)
		}

		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q, want application/json", got)
		}

		payload := map[string]string{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if payload["message"] != "backup complete" || payload["title"] != "Backup done" {
			t.Fatalf("unexpected payload: %#v", payload)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, Token: "secret-token"})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestNotifyReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, Token: "secret-token"})
	if err == nil {
		t.Fatal("expected status error")
	}
}

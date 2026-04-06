package ntfy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestNotifyUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("authorization = %q, want bearer token", got)
		}

		if got := r.Header.Get("Title"); got != "Backup done" {
			t.Fatalf("title = %q, want Backup done", got)
		}

		if got := r.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("content-type = %q, want text/plain", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if string(body) != "backup complete" {
			t.Fatalf("body = %q, want backup complete", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, Token: "secret-token"})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestNotifyUsesBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth credentials")
		}

		if user != "andy" || password != "password" {
			t.Fatalf("basic auth = %q/%q, want andy/password", user, password)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, User: "andy", Password: "password"})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestNotifyRequiresCredentials(t *testing.T) {
	err := Notify("backup complete", types.PushConfig{Url: "https://ntfy.sh/topic"})
	if err == nil {
		t.Fatal("expected credential error")
	}

	if !strings.Contains(err.Error(), "neither user, password and token are set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotifyReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, Token: "secret-token"})
	if err == nil {
		t.Fatal("expected status error")
	}
}

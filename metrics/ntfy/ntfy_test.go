package ntfy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestNotifyAddsEmailHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if got := string(body); got != "backup took 2m" {
			t.Fatalf("unexpected body: %q", got)
		}

		if got := r.Header.Get("Title"); got != "Backup done" {
			t.Fatalf("unexpected title header: %q", got)
		}

		if got := r.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("unexpected content-type header: %q", got)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer my-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		if got := r.Header.Get("Email"); got != "user@example.com" {
			t.Fatalf("unexpected email header: %q", got)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Notify("backup took 2m", types.PushConfig{
		Url:   server.URL,
		Token: "my-token",
		Email: "user@example.com",
	})
	if err != nil {
		t.Fatalf("notify returned error: %v", err)
	}
}

func TestNotifyUsesBasicAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth")
		}

		if user != "andy" || password != "password" {
			t.Fatalf("unexpected basic auth credentials: %q/%q", user, password)
		}

		if got := r.Header.Get("Email"); got != "" {
			t.Fatalf("unexpected email header: %q", got)
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
	t.Parallel()

	err := Notify("backup complete", types.PushConfig{Url: "https://ntfy.sh/topic"})
	if err == nil {
		t.Fatal("expected credential error")
	}

	if !strings.Contains(err.Error(), "neither user, password and token are set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotifyReturnsStatusError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := Notify("backup complete", types.PushConfig{Url: server.URL, Token: "secret-token"})
	if err == nil {
		t.Fatal("expected status error")
	}
}

package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

//nolint:paralleltest // These handlers use Global, which each test temporarily replaces.
func TestRunningTracksOverlappingRuns(t *testing.T) {
	old := Global
	Global = &store{}
	defer func() { Global = old }()
	check := func(want string) {
		t.Helper()
		w := httptest.NewRecorder()
		handleRunning(w, httptest.NewRequest(http.MethodGet, "/api/running", nil))
		if strings.TrimSpace(w.Body.String()) != want {
			t.Fatalf("running = %s, want %s", w.Body.String(), want)
		}
	}
	Global.BeginRun()
	Global.BeginRun()
	check(`{"running":true}`)
	Global.EndRun()
	check(`{"running":true}`)
	w := httptest.NewRecorder()
	handleRun(w, httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader(`{"index":-1}`)))
	if w.Code != http.StatusConflict {
		t.Fatalf("got %d, want conflict", w.Code)
	}
	Global.EndRun()
	check(`{"running":false}`)
}

//nolint:paralleltest // These handlers use Global, which each test temporarily replaces.
func TestAPIRunReservationDoesNotClearScheduledRun(t *testing.T) {
	old := Global
	Global = &store{}
	defer func() { Global = old }()
	finished := make(chan struct{})
	Global.SetRunFunc(func(int) {
		// Simulate a scheduled run starting while the API reservation is held.
		Global.BeginRun()
		close(finished)
	})
	w := httptest.NewRecorder()
	handleRun(w, httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader(`{"index":-1}`)))
	if w.Code != http.StatusAccepted {
		t.Fatalf("got %d, want accepted", w.Code)
	}
	<-finished
	// Wait for the API goroutine to release its reservation.
	deadline := time.Now().Add(time.Second)
	for {
		w = httptest.NewRecorder()
		handleRun(w, httptest.NewRequest(http.MethodPost, "/api/run", nil))
		if w.Code != http.StatusConflict {
			t.Fatalf("active scheduled run lost: %d", w.Code)
		}
		if atomic.LoadInt32(&Global.running) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("API reservation not released")
		}
		time.Sleep(time.Millisecond)
	}
	Global.EndRun()
}

//nolint:paralleltest // These handlers use Global, which each test temporarily replaces.
func TestConfigCreatesMissingFileAndRejectsInvalidDocuments(t *testing.T) {
	old := Global
	Global = &store{}
	defer func() { Global = old }()
	file := filepath.Join(t.TempDir(), "config", "conf.yml")
	Global.SetConfigFiles([]string{file})
	w := httptest.NewRecorder()
	handleConfig(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("missing config: %d %s", w.Code, w.Body.String())
	}
	valid := "source:\n  any:\n    - url: /repo\ndestination:\n  local:\n    - path: /backups\n"
	w = httptest.NewRecorder()
	handleConfig(w, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(valid)))
	if w.Code != http.StatusOK {
		t.Fatalf("create config: %d %s", w.Code, w.Body.String())
	}
	for _, invalid := range []string{"", "# comment only", "{}", valid + "---\nsource: ["} {
		w = httptest.NewRecorder()
		handleConfig(w, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(invalid)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid config accepted: %q (%d)", invalid, w.Code)
		}
		data, err := os.ReadFile(file)
		if err != nil || string(data) != valid {
			t.Fatalf("existing config changed: %q, %v", data, err)
		}
	}
	w = httptest.NewRecorder()
	handleConfig(w, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(valid+"---\n"+valid)))
	if w.Code != http.StatusOK {
		t.Fatalf("multiple valid documents rejected: %s", w.Body.String())
	}
}

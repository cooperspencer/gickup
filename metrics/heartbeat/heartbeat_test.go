package heartbeat

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestSendCallsEachHeartbeatURL(t *testing.T) {
	t.Parallel()

	var hits int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	Send(types.HeartbeatConfig{URLs: []string{server.URL, server.URL}}, true)

	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("hits = %d, want 2", got)
	}
}

func TestSendCallsFailureURLsOnFailure(t *testing.T) {
	t.Parallel()

	var successHits, failureHits int32

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&successHits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer successServer.Close()

	failureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&failureHits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer failureServer.Close()

	Send(types.HeartbeatConfig{
		URLs:        []string{successServer.URL},
		FailureURLs: []string{failureServer.URL},
	}, false)

	if got := atomic.LoadInt32(&successHits); got != 0 {
		t.Fatalf("successHits = %d, want 0", got)
	}
	if got := atomic.LoadInt32(&failureHits); got != 1 {
		t.Fatalf("failureHits = %d, want 1", got)
	}
}

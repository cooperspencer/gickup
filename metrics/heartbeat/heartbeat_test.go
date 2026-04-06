package heartbeat

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestSendCallsEachHeartbeatURL(t *testing.T) {
	var hits int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	Send(types.HeartbeatConfig{URLs: []string{server.URL, server.URL}})

	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("hits = %d, want 2", got)
	}
}

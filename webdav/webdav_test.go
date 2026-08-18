package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"
)

// fakeNetError is a minimal net.Error for table tests.
type fakeNetError struct{ timeout bool }

func (fakeNetError) Error() string   { return "net error" }
func (e fakeNetError) Timeout() bool { return e.timeout }
func (fakeNetError) Temporary() bool { return false }

func TestShouldRetry(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		resp *http.Response
		err  error
		want bool
	}{
		{"503", &http.Response{StatusCode: http.StatusServiceUnavailable}, nil, true},
		{"429", &http.Response{StatusCode: http.StatusTooManyRequests}, nil, true},
		{"500", &http.Response{StatusCode: http.StatusInternalServerError}, nil, true},
		{"502", &http.Response{StatusCode: http.StatusBadGateway}, nil, true},
		{"504", &http.Response{StatusCode: http.StatusGatewayTimeout}, nil, true},
		{"200", &http.Response{StatusCode: http.StatusOK}, nil, false},
		{"404", &http.Response{StatusCode: http.StatusNotFound}, nil, false},
		{"401", &http.Response{StatusCode: http.StatusUnauthorized}, nil, false},
		{"net timeout", nil, fakeNetError{timeout: true}, true},
		{"eof", nil, io.EOF, true},
		{"unexpected eof", nil, io.ErrUnexpectedEOF, true},
		{"conn reset", nil, syscall.ECONNRESET, true},
		{"conn refused", nil, syscall.ECONNREFUSED, true},
		{"broken pipe", nil, syscall.EPIPE, true},
		{"wrapped eof", nil, fmt.Errorf("wrap: %w", io.EOF), true},
		{"context canceled", nil, context.Canceled, false},
		{"context deadline", nil, context.DeadlineExceeded, false},
		{"plain error", nil, errors.New("boom"), false},
		{"non-timeout net", nil, fakeNetError{timeout: false}, false},
	}
	for _, c := range cases {
		if got := shouldRetry(c.resp, c.err); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

// TestRetryAbortsOnContextCancel verifies the backoff sleep yields to a
// canceled request context instead of blocking for the full retry window.
func TestRetryAbortsOnContextCancel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(60*time.Millisecond, cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	resp, err := (&retryRoundTripper{base: http.DefaultTransport}).RoundTrip(req)
	elapsed := time.Since(start)
	if resp != nil {
		resp.Body.Close()
	}

	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("retry did not abort on cancel: %v (backoff base is 500ms)", elapsed)
	}
}

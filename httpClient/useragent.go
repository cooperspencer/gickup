package httpClient

import (
	"net/http"
	"sync"

	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Custom UserAgent for tracking gickup calls in the network
// Some Services may only serve git-over-HTTP endpoints to clients whose User-Agent starts with "git/"
const gickupUserAgent = "git/gickup"

type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", gickupUserAgent)
	return t.base.RoundTrip(clone)
}

// Run once sync used to gate init of the UserAgent for go-git
var installUserAgentOnce sync.Once

// ApplyGickupUserAgent installs go-git HTTP(S) clients that send a git-compatible User-Agent.
// "git/" User-Agent is what the real git client sends, so it is accepted by every other hoster.
func ApplyGickupUserAgent() {
	installUserAgentOnce.Do(func() {
		hc := &http.Client{Transport: userAgentTransport{base: http.DefaultTransport}}
		c := githttp.NewClient(hc)
		gitclient.InstallProtocol("http", c)
		gitclient.InstallProtocol("https", c)
	})
}

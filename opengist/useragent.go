package opengist

import (
	"net/http"
	"sync"

	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Opengist only serves its git-over-HTTP endpoints to clients whose User-Agent starts with "git/"
const gitUserAgent = "git/gickup"

var installUserAgentOnce sync.Once

type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", gitUserAgent)
	return t.base.RoundTrip(clone)
}

// forceGitUserAgent installs go-git HTTP(S) clients that send a git-compatible User-Agent.
// "git/" User-Agent is what the real git client sends, so it is accepted by every other hoster.
func forceGitUserAgent() {
	installUserAgentOnce.Do(func() {
		hc := &http.Client{Transport: userAgentTransport{base: http.DefaultTransport}}
		c := githttp.NewClient(hc)
		gitclient.InstallProtocol("http", c)
		gitclient.InstallProtocol("https", c)
	})
}

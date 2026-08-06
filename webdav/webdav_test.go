package webdav

import (
	"strings"
	"testing"
)

func TestRelKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		href, basePath, want string
	}{
		{"/dav/repos/r/objects/ab/c", "/dav/repos", "r/objects/ab/c"},
		{"https://h/dav/repos/r/HEAD", "/dav/repos", "r/HEAD"},
		{"/dav/repos/r%20name/f", "/dav/repos", "r name/f"},
		{"/dav/repos/r/", "/dav/repos", "r"},
	}
	for _, c := range cases {
		if got := relKey(c.href, c.basePath); got != c.want {
			t.Errorf("relKey(%q, %q) = %q, want %q", c.href, c.basePath, got, c.want)
		}
	}
}

func TestParseMultistatus(t *testing.T) {
	t.Parallel()

	body := `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
  <D:response><D:href>/dav/repos/r/HEAD</D:href>
    <D:propstat><D:prop><D:resourcetype/></D:prop></D:propstat></D:response>
  <D:response><D:href>/dav/repos/r/</D:href>
    <D:propstat><D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop></D:propstat></D:response>
</D:multistatus>`

	res, err := parseMultistatus(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parseMultistatus: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d responses, want 2", len(res))
	}
	if res[0].isCollection() {
		t.Error("HEAD must not be a collection")
	}
	if !res[1].isCollection() {
		t.Error("r/ must be a collection")
	}
}

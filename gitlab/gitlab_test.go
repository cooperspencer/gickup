package gitlab

import (
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestGetRepoVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configured    string
		sourcePrivate bool
		want          gitlab.VisibilityValue
	}{
		{name: "private overrides public source", configured: "private", want: gitlab.PrivateVisibility},
		{name: "public overrides private source", configured: "public", sourcePrivate: true, want: gitlab.PublicVisibility},
		{name: "empty follows private source", sourcePrivate: true, want: gitlab.PrivateVisibility},
		{name: "empty follows public source", want: gitlab.PublicVisibility},
		{name: "source follows private source", configured: "source", sourcePrivate: true, want: gitlab.PrivateVisibility},
		{name: "source follows public source", configured: "source", want: gitlab.PublicVisibility},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := getRepoVisibility(test.configured, test.sourcePrivate); got != test.want {
				t.Fatalf("getRepoVisibility(%q, %t) = %q, want %q", test.configured, test.sourcePrivate, got, test.want)
			}
		})
	}
}

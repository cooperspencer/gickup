package local

import (
	"strings"
	"testing"
)

func TestRandomStringLengthAndCharset(t *testing.T) {
	t.Parallel()

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	got := RandomString(64)
	if len(got) != 64 {
		t.Fatalf("len(RandomString()) = %d, want 64", len(got))
	}

	for _, r := range got {
		if !strings.ContainsRune(charset, r) {
			t.Fatalf("unexpected rune %q in %q", r, got)
		}
	}
}

func TestRandomStringZeroLength(t *testing.T) {
	t.Parallel()

	if got := RandomString(0); got != "" {
		t.Fatalf("RandomString(0) = %q, want empty string", got)
	}
}

package main

import (
	"strings"
	"testing"
)

func TestTildeReplacement_NoAction(t *testing.T) {
	t.Parallel()

	if path := "/boop"; substituteHomeForTildeInPath(path) != path {
		t.Error("Altered path when no alteration was expected")
	}
}

func TestTildeReplacement_TildeOnly(t *testing.T) {
	t.Parallel()

	if path := "~"; substituteHomeForTildeInPath(path) == path {
		t.Error("Path unaltered when alteration was expected")
	}
}

func TestTildeReplacement_TildeDir(t *testing.T) {
	t.Parallel()

	path := "~/boop"
	actual := substituteHomeForTildeInPath(path)
	if strings.HasPrefix(actual, "~") {
		t.Error("Altered path still contains ~")
	}

	if !strings.HasSuffix(actual, "boop") {
		t.Error("Altered path does not end with directory to be retained")
	}
}

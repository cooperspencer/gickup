package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
)

func TestNewRollingFileCreatesDirectory(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "logs")

	writer := NewRollingFile(types.FileLogging{Dir: logDir, File: "gickup.log", MaxAge: 7})
	if writer == nil {
		t.Fatal("expected rolling file writer")
	}

	if info, err := os.Stat(logDir); err != nil || !info.IsDir() {
		t.Fatalf("expected log dir to exist, err=%v", err)
	}
}

//nolint:paralleltest // Mutates package-global exitcode and must remain serial.
func TestErrorHookSetsExitCode(t *testing.T) {
	exitcode = 0
	hook := ErrorHook{}

	hook.Run(nil, zerolog.InfoLevel, "info")
	if got := GetExitCode(); got != 0 {
		t.Fatalf("GetExitCode() after info = %d, want 0", got)
	}

	hook.Run(nil, zerolog.ErrorLevel, "error")
	if got := GetExitCode(); got != 1 {
		t.Fatalf("GetExitCode() after error = %d, want 1", got)
	}
}

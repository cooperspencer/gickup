package logger

import (
	"io"
	"os"
	"path"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	exitcode = 0
)

// NewRollingFile TODO.
func NewRollingFile(config types.FileLogging) io.Writer {
	if config.Dir != "" {
		if err := os.MkdirAll(config.Dir, 0o744); err != nil {
			log.Error().
				Err(err).
				Str("path", config.Dir).
				Msg("can't create log directory")

			return nil
		}
	} else {
		config.Dir = "."
	}

	return &lumberjack.Logger{
		Filename: path.Join(config.Dir, config.File),
		MaxAge:   config.MaxAge, // days
	}
}

// CreateLogger create an instance of Logger.
func CreateLogger(conf types.Logging) zerolog.Logger {
	var writers []io.Writer

	writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: conf.Timeformat})

	if conf.FileLogging.File != "" {
		writers = append(writers, NewRollingFile(conf.FileLogging))
	}

	mw := io.MultiWriter(writers...)

	return zerolog.New(mw).With().Timestamp().Logger()
}

// CreateSubLogger create a sublogger for modules
func CreateSubLogger(args ...string) zerolog.Logger {
	sub := log.With()
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := args[i]
			value := args[i+1]

			// Add key-value pairs to the sublogger
			sub = sub.Str(key, value)
		}
	}
	return sub.Logger()
}

func SetExitCode(code int) {
	exitcode = code
}

func GetExitcode() int {
	return exitcode
}

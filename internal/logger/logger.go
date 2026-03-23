package logger

import (
	"errors"
	"os"
	"time"

	flockerrors "github.com/harshithl1777/flock/internal/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var base *zap.Logger

// init configures the shared zap logger used across the application.
//
// FLOCK_ENV=development uses a human-readable console encoder. Production and
// an empty environment both use JSON output for machine consumption.
func init() {
	base = mustNewLogger(os.Getenv("FLOCK_ENV"), zapcore.AddSync(os.Stdout), true)
}

// mustNewLogger constructs a logger and panics if configuration fails.
func mustNewLogger(env string, sink zapcore.WriteSyncer, colorize bool) *zap.Logger {
	logger, err := newLogger(env, sink, colorize)
	if err != nil {
		panic(err)
	}

	return logger
}

// newLogger builds a zap logger for the configured environment.
func newLogger(env string, sink zapcore.WriteSyncer, colorize bool) (*zap.Logger, error) {
	encoder := newEncoder(env, colorize)
	core := zapcore.NewCore(encoder, sink, zap.InfoLevel)
	if env == "development" {
		core = &messagePaddingCore{Core: core, width: messageWidth}
	}

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), nil
}

// Sync flushes any buffered log entries to their destination.
func Sync() error {
	if base == nil {
		return nil
	}

	return base.Sync()
}

// Info logs an informational message with optional structured fields.
func Info(msg string, fields ...zap.Field) {
	base.Info(msg, fields...)
}

// Error logs an error message with optional structured fields.
func Error(msg string, fields ...zap.Field) {
	base.Error(msg, fields...)
}

// Fatal logs a fatal message with optional structured fields and exits.
func Fatal(msg string, fields ...zap.Field) {
	base.Fatal(msg, fields...)
}

// With returns a child logger with fields attached to every entry.
func With(fields ...zap.Field) *zap.Logger {
	return base.With(fields...)
}

// String constructs a string field for structured logs.
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int constructs an integer field for structured logs.
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Duration constructs a duration field for structured logs.
func Duration(key string, value time.Duration) zap.Field {
	return zap.Duration(key, value)
}

// Any constructs a generic field for structured logs.
func Any(key string, value any) zap.Field {
	return zap.Any(key, value)
}

// Err constructs an error field for structured logs.
func Err(err error) zap.Field {
	var opErr *flockerrors.OpError
	if errors.As(err, &opErr) {
		return zap.Object("error", opErr)
	}

	return zap.Error(err)
}

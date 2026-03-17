package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// init configures the standard logger flags used across the application.
func init() {
	log.SetFlags(log.Ldate | log.Ltime)
}

// caller returns the base file name and line number for the caller.
//
// The skip value follows runtime.Caller semantics.
func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}

	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// Info logs a formatted informational message with caller location metadata.
func Info(format string, args ...any) {
	loc := caller(2)
	log.Printf("%s [INFO] %s", loc, fmt.Sprintf(format, args...))
}

// Error logs a formatted error message with caller location metadata.
func Error(format string, args ...any) {
	loc := caller(2)
	log.Printf("%s [ERROR] %s", loc, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message with caller location metadata.
//
// It exits the process with status code 1 after writing the log entry.
func Fatal(args ...any) {
	loc := caller(2)
	log.Printf("%s [FATAL] %s", loc, fmt.Sprint(args...))
	os.Exit(1)
}

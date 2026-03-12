package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
}

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}

	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func Info(format string, args ...any) {
	loc := caller(2)
	log.Printf("%s [INFO] %s", loc, fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	loc := caller(2)
	log.Printf("%s [ERROR] %s", loc, fmt.Sprintf(format, args...))
}

func Fatal(args ...any) {
	loc := caller(2)
	log.Printf("%s [FATAL] %s", loc, fmt.Sprint(args...))
	os.Exit(1)
}

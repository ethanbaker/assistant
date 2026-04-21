// logger defines slog instance to use across the application
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

var log *slog.Logger

func init() {
	level := slog.LevelInfo

	debug := strings.ToLower(os.Getenv("DEBUG"))
	if debug == "true" || debug == "1" || debug == "yes" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // filename:line
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	log = slog.New(handler)
}

// Static logger functions

func logWithCaller(level slog.Level, msg string) {
	ctx := context.Background()
	if !log.Enabled(ctx, level) {
		return
	}

	// Skip runtime.Callers + logWithCaller + wrapper function to capture user callsite.
	var pcs [1]uintptr
	n := runtime.Callers(3, pcs[:])

	var pc uintptr
	if n > 0 {
		pc = pcs[0]
	}

	rec := slog.NewRecord(time.Now(), level, msg, pc)
	_ = log.Handler().Handle(ctx, rec)
}

func Debugf(format string, args ...any) {
	logWithCaller(slog.LevelDebug, fmt.Sprintf(format, args...))
}

func Debug(msg string) {
	logWithCaller(slog.LevelDebug, msg)
}

func Infof(format string, args ...any) {
	logWithCaller(slog.LevelInfo, fmt.Sprintf(format, args...))
}

func Info(msg string) {
	logWithCaller(slog.LevelInfo, msg)
}

func Warnf(format string, args ...any) {
	logWithCaller(slog.LevelWarn, fmt.Sprintf(format, args...))
}

func Warn(msg string) {
	logWithCaller(slog.LevelWarn, msg)
}

func Errorf(format string, args ...any) {
	logWithCaller(slog.LevelError, fmt.Sprintf(format, args...))
}

func Error(msg string) {
	logWithCaller(slog.LevelError, msg)
}

func Fatalf(format string, args ...any) {
	Errorf(format, args...)
	os.Exit(1)
}

func Fatal(msg string) {
	Error(msg)
	os.Exit(1)
}

// logger defines slog instance to use across the application
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
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

func Debugf(format string, args ...any) {
	if log.Enabled(context.Background(), slog.LevelDebug) {
		log.Debug(fmt.Sprintf(format, args...))
	}
}

func Debug(msg string) {
	if log.Enabled(context.Background(), slog.LevelDebug) {
		log.Debug(msg)
	}
}

func Infof(format string, args ...any) {
	log.Info(fmt.Sprintf(format, args...))
}

func Info(msg string) {
	log.Info(msg)
}

func Warnf(format string, args ...any) {
	log.Warn(fmt.Sprintf(format, args...))
}

func Warn(msg string) {
	log.Warn(msg)
}

func Errorf(format string, args ...any) {
	log.Error(fmt.Sprintf(format, args...))
}

func Error(msg string) {
	log.Error(msg)
}

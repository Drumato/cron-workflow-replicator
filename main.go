package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/drumato/cron-workflow-replicator/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logLevelString := os.Getenv("LOG_LEVEL")
	var logLevel slog.Level

	switch strings.ToLower(logLevelString) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	c := cmd.New()

	if err := c.ExecuteContext(ctx); err != nil {
		slog.ErrorContext(ctx, "execution failed", "error", err)
		os.Exit(1)
	}
}

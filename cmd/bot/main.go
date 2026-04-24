package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mishkamashka/weekly-planner/internal/api"
	"github.com/mishkamashka/weekly-planner/internal/config"
	"github.com/mishkamashka/weekly-planner/internal/runner"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	multi := runner.NewMulti(
		api.NewServer(cfg.HTTPPort),
		// add more runners here: bot, scheduler, etc.
	)

	slog.Info("starting weekly-planner bot")

	if err := multi.Run(ctx); err != nil {
		slog.Error("service exited with error", "err", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}

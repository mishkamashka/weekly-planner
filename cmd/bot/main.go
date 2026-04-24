package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mishkamashka/weekly-planner/internal/api"
	"github.com/mishkamashka/weekly-planner/internal/bot"
	"github.com/mishkamashka/weekly-planner/internal/config"
	"github.com/mishkamashka/weekly-planner/internal/runner"
	"github.com/mishkamashka/weekly-planner/internal/store"
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

	db, err := store.New(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}

	tgBot, err := bot.New(cfg.BotToken, cfg.OwnerTelegramID, db)
	if err != nil {
		slog.Error("failed to create bot", "err", err)
		os.Exit(1)
	}

	multi := runner.NewMulti(
		api.NewServer(cfg.HTTPPort),
		tgBot,
	)

	slog.Info("starting weekly-planner bot")

	if err := multi.Run(ctx); err != nil {
		slog.Error("service exited with error", "err", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}

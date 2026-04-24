package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Sandbox         bool
	BotToken        string
	DatabasePath    string
	OwnerTelegramID int64
	HTTPPort        int
	ShutdownTimeout time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	cfg.Sandbox = os.Getenv("SANDBOX") == "true"

	cfg.BotToken = os.Getenv("BOT_TOKEN")
	if cfg.BotToken == "" && !cfg.Sandbox {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	cfg.DatabasePath = os.Getenv("DATABASE_PATH")
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "bot.db"
	}

	rawID := os.Getenv("OWNER_TELEGRAM_ID")
	if rawID == "" && !cfg.Sandbox {
		return nil, fmt.Errorf("OWNER_TELEGRAM_ID is required")
	}
	if rawID != "" {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("OWNER_TELEGRAM_ID must be an integer: %w", err)
		}
		cfg.OwnerTelegramID = id
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		cfg.HTTPPort = 8080
	} else {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("HTTP_PORT must be an integer: %w", err)
		}
		cfg.HTTPPort = p
	}

	timeoutSec := os.Getenv("SHUTDOWN_TIMEOUT_SEC")
	if timeoutSec == "" {
		cfg.ShutdownTimeout = 10 * time.Second
	} else {
		sec, err := strconv.Atoi(timeoutSec)
		if err != nil {
			return nil, fmt.Errorf("SHUTDOWN_TIMEOUT_SEC must be an integer: %w", err)
		}
		cfg.ShutdownTimeout = time.Duration(sec) * time.Second
	}

	return cfg, nil
}

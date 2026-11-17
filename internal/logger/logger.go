package logger

import (
	"log/slog"
	"os"
)

const envProd = "prod"

func New(env string) *slog.Logger {
	switch env {
	case envProd:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

	default:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}))
	}
}

package logx

import (
	"log/slog"
	"os"
)

// New returns the project structured logger.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

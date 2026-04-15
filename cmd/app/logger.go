package app

import (
	"log/slog"
	"os"
	"os/user"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/cmd/app/consts"
)

func initLogger(level string) {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	setDefaultLogger(slog.New(handler))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setDefaultLogger(base *slog.Logger) {
	attrs := []any{
		slog.Int("pid", os.Getpid()),
		slog.String("app", consts.APPNAME),
		slog.String("version", consts.Version),
	}

	if consts.Commit != "" {
		attrs = append(attrs, slog.String("commit", consts.Commit))
	}
	if consts.BuildDate != "" {
		attrs = append(attrs, slog.String("build_date", consts.BuildDate))
	}
	if currentUser, err := user.Current(); err == nil {
		attrs = append(attrs, slog.String("user", currentUser.Username))
	}

	slog.SetDefault(base.With(attrs...))
}

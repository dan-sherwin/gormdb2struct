package generator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openPostgresDB(ctx context.Context, logger *slog.Logger, cfg config.Config) (*gorm.DB, error) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("Connecting to PostgreSQL",
		slog.String("host", cfg.DbHost),
		slog.Int("port", cfg.DbPort),
		slog.String("db", cfg.DbName),
	)

	db, err := gorm.Open(postgres.Open(postgresDSN(cfg)), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL sql.DB handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping PostgreSQL database: %w", err)
	}

	return db, nil
}

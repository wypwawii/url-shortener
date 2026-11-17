package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wypwawii/url-shortener/internal/config"
)

func Connect(ctx context.Context, cfg *config.AppConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.ConnConfig.Host = cfg.Postgres.Host
	poolConfig.ConnConfig.Port = uint16(cfg.Postgres.Port)
	poolConfig.ConnConfig.User = cfg.Postgres.User
	poolConfig.ConnConfig.Password = cfg.Postgres.Password
	poolConfig.ConnConfig.Database = cfg.Postgres.Database

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return pool, nil
}

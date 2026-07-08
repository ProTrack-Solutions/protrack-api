package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	*pgxpool.Pool
}

func NewConnect(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	if config.ConnConfig.Database != cfg.DBName {
		config.ConnConfig.Database = cfg.DBName
	}

	config.MaxConns = 25                     // Máximo de conexões simultaneas no poll
	config.MinConns = 5                      // Minimo de conexões simultaneas no poll
	config.MaxConnLifetime = 5 * time.Minute // Tempo máximo de vida de uma conexão

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool to database '%s': %w", cfg.DBName, err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	URL             string
	ConnMaxLifetime time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
}

func NewDBConfig(url string) *DBConfig {
	return &DBConfig{
		URL:             url,
		ConnMaxLifetime: 10 * time.Minute,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
	}
}

func (cfg *DBConfig) Connect(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	_, err = db.ExecContext(ctx, "SET statement_timeout = '300s'")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set statement timeout: %w", err)
	}

	return db, nil
}

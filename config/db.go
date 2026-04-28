package config

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB(connectionString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	return db, nil
}

func CreateTable(db *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		short_code TEXT UNIQUE NOT NULL,
		long_url TEXT NOT NULL,
		clicks INT DEFAULT 0,
		ip_address TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		expiry TIMESTAMPTZ NULL
	);
	CREATE INDEX IF NOT EXISTS idx_urls_expiry ON urls(expiry) WHERE expiry IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_urls_ip ON urls(ip_address);
	CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
	`
	_, err := db.Exec(context.Background(), query)
	if err != nil {
		return err
	}

	return nil
}

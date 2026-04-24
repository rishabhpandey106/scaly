package main

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
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_code TEXT UNIQUE NOT NULL,
		long_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		expiry TIMESTAMP NULL
	);
	CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
	`
	_, err := db.Exec(context.Background(), query)
	return err
}

func SaveURL(db *pgxpool.Pool, shortCode, longURL string) error {
	query := `
	INSERT INTO urls (short_code, long_url)
	VALUES ($1, $2)
	ON CONFLICT (short_code) DO UPDATE SET long_url = EXCLUDED.long_url
	`
	_, err := db.Exec(context.Background(), query, shortCode, longURL)
	return err
}

func GetURL(db *pgxpool.Pool, shortCode string) (string, error) {
	var longURL string
	query := `SELECT long_url FROM urls WHERE short_code = $1`
	err := db.QueryRow(context.Background(), query, shortCode).Scan(&longURL)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepo struct {
	DB *pgxpool.Pool
}

func NewURLRepo(db *pgxpool.Pool) *URLRepo {
	return &URLRepo{DB: db}
}

func (r *URLRepo) Save(shortCode, longURL string) error {
	_, err := r.DB.Exec(context.Background(),
		"INSERT INTO urls (short_code, long_url) VALUES ($1, $2)",
		shortCode, longURL,
	)
	return err
}

func (r *URLRepo) Get(shortCode string) (string, error) {
	var url string

	err := r.DB.QueryRow(context.Background(),
		"SELECT long_url FROM urls WHERE short_code=$1",
		shortCode,
	).Scan(&url)

	return url, err
}

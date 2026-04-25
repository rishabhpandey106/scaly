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

func (r *URLRepo) GetWithClicks(code string) (string, int, error) {
	var url string
	var clicks int

	query := `SELECT long_url, clicks FROM urls WHERE short_code=$1`

	err := r.DB.QueryRow(context.Background(), query, code).Scan(&url, &clicks)
	if err != nil {
		return "", 0, err
	}

	return url, clicks, nil
}

func (r *URLRepo) IncrementClicks(code string) {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code=$1`
	r.DB.Exec(context.Background(), query, code)
}

func (r *URLRepo) UpdateClicks(code string, clicks int) {
	query := `UPDATE urls SET clicks = clicks + $1 WHERE short_code=$2`
	r.DB.Exec(context.Background(), query, clicks, code)
}

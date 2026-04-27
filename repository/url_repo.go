package repository

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepo struct {
	DB *pgxpool.Pool
}

func NewURLRepo(db *pgxpool.Pool) *URLRepo {
	return &URLRepo{DB: db}
}

func (r *URLRepo) Save(code string, url string, expiry *time.Time, ip string) error {
	query := `INSERT INTO urls (short_code, long_url, expiry, ip_address) VALUES ($1, $2, $3, $4)`
	_, err := r.DB.Exec(context.Background(), query, code, url, expiry, ip)
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

func (r *URLRepo) GetWithClicks(code string) (string, int, *time.Time, error) {
	var url string
	var clicks int
	var expiry *time.Time

	query := `SELECT long_url, clicks, expiry FROM urls WHERE short_code=$1`

	err := r.DB.QueryRow(context.Background(), query, code).Scan(&url, &clicks, &expiry)
	if err != nil {
		return "", 0, nil, err
	}

	return url, clicks, expiry, nil
}

func (r *URLRepo) IncrementClicks(code string) {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code=$1`
	r.DB.Exec(context.Background(), query, code)
}

func (r *URLRepo) UpdateClicks(code string, clicks int) {
	query := `UPDATE urls SET clicks = clicks + $1 WHERE short_code=$2`
	r.DB.Exec(context.Background(), query, clicks, code)
}

func (r *URLRepo) DeleteExpired() error {
	query := `DELETE FROM urls WHERE expiry IS NOT NULL AND expiry < NOW()`
	result, err := r.DB.Exec(context.Background(), query)
	if err != nil {
		return err
	}

	rows := result.RowsAffected()
	log.Printf("expired urls deleted: %d", rows)
	return nil
}

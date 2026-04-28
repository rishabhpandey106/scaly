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

func (r *URLRepo) Save(code string, url string, expiry *time.Time, ip string, userID int) error {
	query := `INSERT INTO urls (short_code, long_url, expiry, ip_address, user_id) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.DB.Exec(context.Background(), query, code, url, expiry, ip, userID)
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

func (r *URLRepo) GetUserURLs(userID int) ([]string, error) {
	var shortCodes []string
	
	query := `SELECT short_code FROM urls WHERE user_id = $1`
	
	rows, err := r.DB.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		shortCodes = append(shortCodes, code)
	}
	
	return shortCodes, nil
}

func (r *URLRepo) DeleteURL(code string, userID int) error {
	query := `DELETE FROM urls WHERE short_code=$1 AND user_id=$2`
	_, err := r.DB.Exec(context.Background(), query, code, userID)
	return err
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

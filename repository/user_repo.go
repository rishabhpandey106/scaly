package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	DB *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{DB: db}
}

func (r *UserRepo) Create(email, hashedPassword string) (int, error) {
	var id int
	query := `INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`
	err := r.DB.QueryRow(context.Background(), query, email, hashedPassword).Scan(&id)
	return id, err
}

func (r *UserRepo) GetByEmail(email string) (int, string, error) {
	var id int
	var password string
	query := `SELECT id, password FROM users WHERE email = $1`
	err := r.DB.QueryRow(context.Background(), query, email).Scan(&id, &password)
	return id, password, err
}

func (r *UserRepo) GetByID(id int) (string, error) {
	var email string
	query := `SELECT email FROM users WHERE id = $1`
	err := r.DB.QueryRow(context.Background(), query, id).Scan(&email)
	return email, err
}
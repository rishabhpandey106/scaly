package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email" validate:"required,email"`
	Password  string    `json:"-" validate:"required,min=8"`
	CreatedAt time.Time `json:"createdAt"`
}
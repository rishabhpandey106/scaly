package model

import "time"

type URL struct {
	ID        int
	UserID    int
	ShortCode string
	LongURL   string
	CreatedAt time.Time
}

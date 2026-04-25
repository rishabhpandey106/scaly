package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

type URLService struct {
	Repo interface {
		Save(string, string) error
		Get(string) (string, error)
		GetWithClicks(string) (string, int, error)
		IncrementClicks(string)
		UpdateClicks(string, int)
	}
	Cache *redis.Client
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var ctx = context.Background()

func generateCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func NewURLService(repo interface {
	Save(string, string) error
	Get(string) (string, error)
	GetWithClicks(string) (string, int, error)
	IncrementClicks(string)
	UpdateClicks(string, int)
}, Cache *redis.Client) *URLService {
	return &URLService{Repo: repo, Cache: Cache}
}

func (s *URLService) CreateURL(longURL string) (string, error) {
	code := generateCode(8)

	err := s.Repo.Save(code, longURL)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (s *URLService) GetURL(code string) (string, error) {
	key := "url:" + code
	clickKey := "url:" + code + ":clicks"

	val, err := s.Cache.Get(ctx, key).Result()
	if err == nil {
		// go s.Repo.IncrementClicks(code) // async increment clicks in background : goroutine
		s.Cache.Incr(ctx, clickKey)
		return val, nil
	}

	// url, err := s.Repo.Get(code)
	url, clicks, err := s.Repo.GetWithClicks(code)
	if err != nil {
		return "", err
	}

	// go s.Repo.IncrementClicks(code)
	ttl := 5 * time.Minute
	if clicks > 100 {
		ttl = 1 * time.Hour
	}
	s.Cache.Set(ctx, key, url, ttl)

	s.Cache.SetArgs(ctx, clickKey, 0, redis.SetArgs{
		Mode: "NX",
	})
	s.Cache.Incr(ctx, clickKey)
	return url, nil
}

func (s *URLService) CheckAlias(code string) (bool, error) {

	val, err := s.Repo.Get(code)
	if err != nil {
		return false, nil // not found - available
	}

	if val == "" {
		return false, nil
	}

	return true, nil // exists - not available
}

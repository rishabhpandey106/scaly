package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
	"url-shortener/utils"

	"github.com/redis/go-redis/v9"
)

type URLService struct {
	Repo interface {
		Save(string, string, *time.Time, string) error
		Get(string) (string, error)
		GetWithClicks(string) (string, int, *time.Time, error)
		IncrementClicks(string)
		UpdateClicks(string, int)
	}
	Cache *redis.Client
}

var ctx = context.Background()

func NewURLService(repo interface {
	Save(string, string, *time.Time, string) error
	Get(string) (string, error)
	GetWithClicks(string) (string, int, *time.Time, error)
	IncrementClicks(string)
	UpdateClicks(string, int)
}, Cache *redis.Client) *URLService {
	return &URLService{Repo: repo, Cache: Cache}
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
	url, _, expiry, err := s.Repo.GetWithClicks(code)
	if err != nil {
		return "", err
	}

	if expiry != nil && time.Now().UTC().After(*expiry) {
		return "", errors.New("link expired")
	}

	// go s.Repo.IncrementClicks(code)
	ttl := 5 * time.Minute
	if expiry != nil {
		remaining := time.Until(*expiry)
		log.Printf("URL %s expires in %s\n", code, remaining)

		if remaining <= 0 {
			return "", errors.New("link expired")
		}

		if remaining < ttl {
			ttl = remaining
		}
	}

	// if clicks > 100 {
	// 	ttl = 1 * time.Hour
	// }

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

func (s *URLService) CreateURL(longURL string, alias *string, expiry *time.Time, ip string) (string, error) {

	longURL = strings.TrimSpace(longURL)
	if longURL == "" {
		return "", errors.New("url is required")
	}

	if expiry != nil && time.Now().UTC().After(*expiry) {
		return "", errors.New("expiry must be in future")
	}

	// custom alias flow
	if alias != nil && *alias != "" {

		exists, _ := s.Repo.Get(*alias)
		if exists != "" {
			return "", errors.New("alias already exists")
		}

		err := s.Repo.Save(*alias, longURL, expiry, ip)
		if err != nil {
			return "", err
		}

		key := "url:" + *alias
		if expiry != nil {
			ttl := time.Until(*expiry)
			if ttl > 0 {
				s.Cache.Set(ctx, key, longURL, ttl)
			}
		}

		return *alias, nil
	}
	// redis counter flow
	counter, err := s.Cache.Incr(ctx, "global:counter").Result()
	if err != nil {
		return "", err
	}

	code := utils.ToBase62(counter)

	// safety check (rare collision)
	for {
		exists, _ := s.Repo.Get(code)
		if exists == "" {
			break
		}

		counter, _ = s.Cache.Incr(ctx, "global:counter").Result()
		code = utils.ToBase62(counter)
	}

	err = s.Repo.Save(code, longURL, expiry, ip)
	if err != nil {
		return "", err
	}

	key := "url:" + code

	if expiry != nil {
		ttl := time.Until(*expiry)
		if ttl > 0 {
			s.Cache.Set(ctx, key, longURL, ttl)
		}
	}

	return code, nil
}

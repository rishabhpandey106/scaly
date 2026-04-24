package service

import (
	"math/rand"
)

type URLService struct {
	Repo interface {
		Save(string, string) error
		Get(string) (string, error)
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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
}) *URLService {
	return &URLService{Repo: repo}
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
	return s.Repo.Get(code)
}

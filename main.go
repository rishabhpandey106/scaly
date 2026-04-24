package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

const (
	shortCodeLength = 8
	minCodeLength   = 6
)

var (
	baseURL = "http://localhost:8000"
	storage = make(map[string]string)
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func generateShortCode() string {
	length := minCodeLength + rand.Intn(shortCodeLength-minCodeLength+1)
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func generateUniqueCode() string {
	for i := 0; i < 100; i++ {
		code := generateShortCode()
		if _, exists := storage[code]; !exists {
			return code
		}
	}
	return generateShortCode()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	app := fiber.New()
	app.Use(logger.New())

	app.Post("/shorten", func(c *fiber.Ctx) error {
		var req ShortenRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(ErrorResponse{Error: "invalid JSON"})
		}

		req.URL = strings.TrimSpace(req.URL)
		if req.URL == "" {
			return c.Status(400).JSON(ErrorResponse{Error: "url is required"})
		}

		code := generateUniqueCode()
		storage[code] = req.URL

		return c.Status(201).JSON(ShortenResponse{
			ShortCode: code,
			ShortURL:  baseURL + "/" + code,
		})
	})

	app.Get("/:code", func(c *fiber.Ctx) error {
		code := c.Params("code")

		url, exists := storage[code]
		if !exists {
			return c.Status(404).JSON(ErrorResponse{Error: "short code not found"})
		}

		return c.Redirect(url, 302)
	})

	app.Listen(":8000")
}

package main

import (
	"math/rand"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateCode(length int) string {
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func ShortenHandler(c *fiber.Ctx) error {

	db := c.Locals("db").(*pgxpool.Pool)

	var req ShortenRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid JSON"})
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "url is required"})
	}

	var code string

	for i := 0; i < 100; i++ {
		code = generateCode(8)

		var exists string
		err := db.QueryRow(c.Context(),
			"SELECT short_code FROM urls WHERE short_code=$1",
			code,
		).Scan(&exists)

		if err != nil {
			break
		}
	}

	if err := SaveURL(db, code, req.URL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to save URL"})
	}

	baseURL := c.BaseURL()

	resp := ShortenResponse{
		ShortCode: code,
		ShortURL:  baseURL + "/" + code,
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func RedirectHandler(c *fiber.Ctx) error {

	db := c.Locals("db").(*pgxpool.Pool)

	code := c.Params("code")

	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "short code is required",
		})
	}

	longURL, err := GetURL(db, code)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "short code not found",
		})
	}

	return c.Redirect(longURL, fiber.StatusFound)
}

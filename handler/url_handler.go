package handler

import (
	"github.com/gofiber/fiber/v2"
)

type URLHandler struct {
	Service interface {
		CreateURL(string) (string, error)
		GetURL(string) (string, error)
	}
}

func NewURLHandler(svc interface {
	CreateURL(string) (string, error)
	GetURL(string) (string, error)
}) *URLHandler {
	return &URLHandler{Service: svc}
}

func (h *URLHandler) Shorten(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}

	code, err := h.Service.CreateURL(req.URL)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed"})
	}

	return c.JSON(fiber.Map{
		"short_code": code,
		"short_url":  c.BaseURL() + "/" + code,
	})
}

func (h *URLHandler) Redirect(c *fiber.Ctx) error {
	code := c.Params("code")

	url, err := h.Service.GetURL(code)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	return c.Redirect(url, 302)
}

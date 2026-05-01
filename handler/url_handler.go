package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type URLHandler struct {
	Service interface {
		CreateURL(string, *string, *time.Time, string, int) (string, error)
		GetURL(string) (string, error)
		CheckAlias(string) (bool, error)
		GetUserURLs(int) ([]string, error)
		DeleteURL(string, int) error
	}
}

func NewURLHandler(svc interface {
	CreateURL(string, *string, *time.Time, string, int) (string, error)
	GetURL(string) (string, error)
	CheckAlias(string) (bool, error)
	GetUserURLs(int) ([]string, error)
	DeleteURL(string, int) error
}) *URLHandler {
	return &URLHandler{Service: svc}
}

func (h *URLHandler) Shorten(c *fiber.Ctx) error {
	var req struct {
		URL    string     `json:"url"`
		Alias  *string    `json:"alias,omitempty"`
		Expiry *time.Time `json:"expiry,omitempty"`
	}

	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	ip := c.IP()
	// log.Printf("ip address: %s", ip)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}

	code, err := h.Service.CreateURL(req.URL, req.Alias, req.Expiry, ip, userID.(int))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"short_code": code,
		"short_url":  c.BaseURL() + "/" + code,
	})
}

func (h *URLHandler) Redirect(c *fiber.Ctx) error {
	// redirect endpoint doesn't require authentication for public access
	// but we could add click tracking for authenticated users, not needed for now
	code := c.Params("code")

	url, err := h.Service.GetURL(code)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	return c.Redirect(url, 302)
}

func (h *URLHandler) CheckAlias(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	code := c.Params("code")

	exists, err := h.Service.CheckAlias(code)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "server error"})
	}

	return c.JSON(fiber.Map{
		"available": !exists,
	})
}

func (h *URLHandler) GetUserURLs(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	urls, err := h.Service.GetUserURLs(userID.(int))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"urls": urls,
	})
}

func (h *URLHandler) DeleteURL(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	code := c.Params("code")
	if code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "URL code is required"})
	}

	err := h.Service.DeleteURL(code, userID.(int))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "URL deleted successfully",
	})
}

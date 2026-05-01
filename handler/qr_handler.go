package handler

import (
	"time"

	"url-shortener/service"

	"github.com/gofiber/fiber/v2"
)

type QRHandler struct {
	URLService interface {
		CreateURL(string, *string, *time.Time, string, int) (string, error)
	}
	QRService *service.QRService
}

func NewQRHandler(svc interface {
	CreateURL(string, *string, *time.Time, string, int) (string, error)
}, qr *service.QRService) *QRHandler {
	return &QRHandler{URLService: svc, QRService: qr}
}

func (h *QRHandler) GenerateQR(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}

	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	code, err := h.URLService.CreateURL(req.URL, nil, nil, c.IP(), userID.(int))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// log.Printf("Generated short code: %s for URL: %s", code, req.URL)
	shortURL := c.BaseURL() + "/" + code

	qr, err := h.QRService.Generate(shortURL)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "QR generation failed"})
	}

	c.Set("Content-Type", "image/png")
	return c.Send(qr)
}

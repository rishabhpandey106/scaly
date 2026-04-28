package handler

import (
	"github.com/gofiber/fiber/v2"

	"url-shortener/service"
)

type AuthHandler struct {
	AuthService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

type SignupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	UserID int    `json:"userId"`
	Token  string `json:"token"`
}

func (h *AuthHandler) Signup(c *fiber.Ctx) error {
	var req SignupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}

	userID, token, err := h.AuthService.Signup(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email already registered" {
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(AuthResponse{
		UserID: userID,
		Token:  token,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}

	userID, token, err := h.AuthService.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	return c.JSON(AuthResponse{
		UserID: userID,
		Token:  token,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// With JWT, logout is typically handled client-side by removing the token
	// future work : implement token blacklisting if needed
	return c.JSON(fiber.Map{"message": "logged out successfully"})
}

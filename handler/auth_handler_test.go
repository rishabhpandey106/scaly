package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/service"
)

// mockAuthService is a test-double for *service.AuthService.
// Because AuthHandler embeds the concrete *service.AuthService we create a
// real AuthService backed by a lightweight mock user-repo.
func newTestAuthHandler() (*AuthHandler, *service.AuthService) {
	repo := &fakeUserRepo{}
	svc := service.NewAuthService(repo, "test-secret", 24*time.Hour)
	return NewAuthHandler(svc), svc
}

// fakeUserRepo implements service.UserRepository without a real DB.
type fakeUserRepo struct {
	users map[string]struct {
		id       int
		password string
	}
	createErr error
}

func (f *fakeUserRepo) ensureInit() {
	if f.users == nil {
		f.users = make(map[string]struct {
			id       int
			password string
		})
	}
}

func (f *fakeUserRepo) Create(email, hash string) (int, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.ensureInit()
	id := len(f.users) + 1
	f.users[email] = struct {
		id       int
		password string
	}{id: id, password: hash}
	return id, nil
}

func (f *fakeUserRepo) GetByEmail(email string) (int, string, error) {
	f.ensureInit()
	u, ok := f.users[email]
	if !ok {
		return 0, "", errors.New("not found")
	}
	return u.id, u.password, nil
}

func newAuthApp(h *AuthHandler) *fiber.App {
	app := fiber.New()
	app.Post("/auth/signup", h.Signup)
	app.Post("/auth/login", h.Login)
	app.Post("/auth/logout", h.Logout)
	return app
}

func jsonBody(v interface{}) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// ---------------------------------------------------------------------------
// Signup
// ---------------------------------------------------------------------------

func TestAuthSignup_Success(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/signup",
		jsonBody(map[string]string{"email": "alice@example.com", "password": "pass1234"}))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result AuthResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Positive(t, result.UserID)
	assert.NotEmpty(t, result.Token)
}

func TestAuthSignup_DuplicateEmail(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	body := jsonBody(map[string]string{"email": "bob@example.com", "password": "pass1234"})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// second signup with same email
	body = jsonBody(map[string]string{"email": "bob@example.com", "password": "other"})
	req = httptest.NewRequest(http.MethodPost, "/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestAuthSignup_InvalidJSON(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, 400, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestAuthLogin_Success(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	// register first
	req := httptest.NewRequest(http.MethodPost, "/auth/signup",
		jsonBody(map[string]string{"email": "carol@example.com", "password": "pass1234"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// now login
	req = httptest.NewRequest(http.MethodPost, "/auth/login",
		jsonBody(map[string]string{"email": "carol@example.com", "password": "pass1234"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result AuthResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
}

func TestAuthLogin_WrongPassword(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/signup",
		jsonBody(map[string]string{"email": "dave@example.com", "password": "pass1234"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	req = httptest.NewRequest(http.MethodPost, "/auth/login",
		jsonBody(map[string]string{"email": "dave@example.com", "password": "wrongpass"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthLogin_UnknownEmail(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		jsonBody(map[string]string{"email": "nobody@example.com", "password": "pass"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthLogin_InvalidJSON(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, 400, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestAuthLogout_AlwaysOK(t *testing.T) {
	h, _ := newTestAuthHandler()
	app := newAuthApp(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "logged out successfully", result["message"])
}

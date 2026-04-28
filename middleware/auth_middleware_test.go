package middleware

import (
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

// fakeUserRepo satisfies service.UserRepository without a real database.
type fakeUserRepo struct {
	users map[string]struct {
		id       int
		password string
	}
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

func newTestAuthService() *service.AuthService {
	return service.NewAuthService(&fakeUserRepo{}, "middleware-secret", time.Hour)
}

func newAuthMiddlewareApp(authSvc *service.AuthService) *fiber.App {
	app := fiber.New()
	app.Use(AuthMiddleware(authSvc))
	app.Get("/protected", func(c *fiber.Ctx) error {
		uid := c.Locals("userID")
		return c.JSON(fiber.Map{"userID": uid})
	})
	return app
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	app := newAuthMiddlewareApp(newTestAuthService())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidFormat_NoBearer(t *testing.T) {
	app := newAuthMiddlewareApp(newTestAuthService())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "sometoken")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidFormat_WrongScheme(t *testing.T) {
	app := newAuthMiddlewareApp(newTestAuthService())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	app := newAuthMiddlewareApp(newTestAuthService())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	authSvc := newTestAuthService()
	app := newAuthMiddlewareApp(authSvc)

	// obtain a real token via Signup
	_, token, err := authSvc.Signup("user@example.com", "pass1234")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_BearerCaseInsensitive(t *testing.T) {
	authSvc := newTestAuthService()
	app := newAuthMiddlewareApp(authSvc)

	_, token, err := authSvc.Signup("user2@example.com", "pass1234")
	require.NoError(t, err)

	// "BEARER" in uppercase should still be accepted
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "BEARER "+token)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_TokenFromWrongSecret(t *testing.T) {
	authSvc := newTestAuthService()
	// AuthService with a different secret issues the token
	otherSvc := service.NewAuthService(&fakeUserRepo{}, "other-secret", time.Hour)

	app := newAuthMiddlewareApp(authSvc)

	_, token, err := otherSvc.Signup("hacker@example.com", "pass1234")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

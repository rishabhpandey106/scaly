package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockURLService implements the URLHandler service interface.
type mockURLService struct {
	createURL   func(string, *string, *time.Time, string, int) (string, error)
	getURL      func(string) (string, error)
	checkAlias  func(string) (bool, error)
	getUserURLs func(int) ([]string, error)
	deleteURL   func(string, int) error
}

func (m *mockURLService) CreateURL(u string, alias *string, exp *time.Time, ip string, uid int) (string, error) {
	return m.createURL(u, alias, exp, ip, uid)
}
func (m *mockURLService) GetURL(code string) (string, error) { return m.getURL(code) }
func (m *mockURLService) CheckAlias(code string) (bool, error) {
	return m.checkAlias(code)
}
func (m *mockURLService) GetUserURLs(uid int) ([]string, error) { return m.getUserURLs(uid) }
func (m *mockURLService) DeleteURL(code string, uid int) error  { return m.deleteURL(code, uid) }

// newURLTestApp wires a fresh Fiber app with the given mock service and seeds
// c.Locals("userID") via the optional userID parameter (0 = unauthenticated).
func newURLTestApp(svc *mockURLService, userID int) *fiber.App {
	app := fiber.New()
	h := NewURLHandler(svc)

	injectUser := func(c *fiber.Ctx) error {
		if userID != 0 {
			c.Locals("userID", userID)
		}
		return c.Next()
	}

	app.Post("/shorten", injectUser, h.Shorten)
	app.Get("/:code", injectUser, h.Redirect)
	app.Get("/alias/check/:code", injectUser, h.CheckAlias)
	app.Get("/user/urls", injectUser, h.GetUserURLs)
	app.Delete("/:code", injectUser, h.DeleteURL)
	return app
}

func doRequest(app *fiber.App, method, path string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	return resp
}

// ---------------------------------------------------------------------------
// Shorten handler
// ---------------------------------------------------------------------------

func TestShorten_Unauthorized(t *testing.T) {
	svc := &mockURLService{}
	app := newURLTestApp(svc, 0) // no userID injected

	resp := doRequest(app, http.MethodPost, "/shorten", map[string]string{"url": "https://example.com"})
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestShorten_Success(t *testing.T) {
	svc := &mockURLService{
		createURL: func(u string, _ *string, _ *time.Time, _ string, _ int) (string, error) {
			return "abc123", nil
		},
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodPost, "/shorten", map[string]string{"url": "https://example.com"})
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "abc123", result["short_code"])
}

func TestShorten_ServiceError(t *testing.T) {
	svc := &mockURLService{
		createURL: func(_ string, _ *string, _ *time.Time, _ string, _ int) (string, error) {
			return "", errors.New("some error")
		},
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodPost, "/shorten", map[string]string{"url": "https://example.com"})
	assert.Equal(t, 500, resp.StatusCode)
}

func TestShorten_InvalidJSON(t *testing.T) {
	svc := &mockURLService{}
	app := newURLTestApp(svc, 1)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	assert.Equal(t, 400, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Redirect handler
// ---------------------------------------------------------------------------

func TestRedirect_Success(t *testing.T) {
	svc := &mockURLService{
		getURL: func(code string) (string, error) { return "https://long.com", nil },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/abc", nil)
	assert.Equal(t, 302, resp.StatusCode)
	assert.Equal(t, "https://long.com", resp.Header.Get("Location"))
}

func TestRedirect_NotFound(t *testing.T) {
	svc := &mockURLService{
		getURL: func(code string) (string, error) { return "", errors.New("not found") },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/missing", nil)
	assert.Equal(t, 404, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// CheckAlias handler
// ---------------------------------------------------------------------------

func TestCheckAlias_Unauthorized(t *testing.T) {
	svc := &mockURLService{}
	app := newURLTestApp(svc, 0)

	resp := doRequest(app, http.MethodGet, "/alias/check/mycode", nil)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestCheckAlias_Available(t *testing.T) {
	svc := &mockURLService{
		checkAlias: func(code string) (bool, error) { return false, nil },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/alias/check/mycode", nil)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]bool
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.True(t, result["available"])
}

func TestCheckAlias_Taken(t *testing.T) {
	svc := &mockURLService{
		checkAlias: func(code string) (bool, error) { return true, nil },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/alias/check/taken", nil)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]bool
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.False(t, result["available"])
}

func TestCheckAlias_ServiceError(t *testing.T) {
	svc := &mockURLService{
		checkAlias: func(code string) (bool, error) { return false, errors.New("db error") },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/alias/check/bad", nil)
	assert.Equal(t, 500, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GetUserURLs handler
// ---------------------------------------------------------------------------

func TestGetUserURLs_Unauthorized(t *testing.T) {
	svc := &mockURLService{}
	app := newURLTestApp(svc, 0)

	resp := doRequest(app, http.MethodGet, "/user/urls", nil)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestGetUserURLs_Success(t *testing.T) {
	svc := &mockURLService{
		getUserURLs: func(uid int) ([]string, error) { return []string{"a1", "b2"}, nil },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/user/urls", nil)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string][]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.ElementsMatch(t, []string{"a1", "b2"}, result["urls"])
}

func TestGetUserURLs_ServiceError(t *testing.T) {
	svc := &mockURLService{
		getUserURLs: func(uid int) ([]string, error) { return nil, errors.New("db error") },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodGet, "/user/urls", nil)
	assert.Equal(t, 500, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// DeleteURL handler
// ---------------------------------------------------------------------------

func TestDeleteURL_Unauthorized(t *testing.T) {
	svc := &mockURLService{}
	app := newURLTestApp(svc, 0)

	resp := doRequest(app, http.MethodDelete, "/abc", nil)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteURL_Success(t *testing.T) {
	svc := &mockURLService{
		deleteURL: func(code string, uid int) error { return nil },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodDelete, "/abc", nil)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "URL deleted successfully", result["message"])
}

func TestDeleteURL_ServiceError(t *testing.T) {
	svc := &mockURLService{
		deleteURL: func(code string, uid int) error { return errors.New("not found") },
	}
	app := newURLTestApp(svc, 1)

	resp := doRequest(app, http.MethodDelete, "/abc", nil)
	assert.Equal(t, 500, resp.StatusCode)
}

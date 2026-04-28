package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRateLimiterApp(rdb *redis.Client) *fiber.App {
	app := fiber.New()
	app.Use(RateLimiter(rdb))
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})
	return app
}

func TestRateLimiter_UnderLimit(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newRateLimiterApp(rdb)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "request %d should succeed", i+1)
	}
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newRateLimiterApp(rdb)

	var lastStatus int
	// Send 105 requests; at least one should return 429.
	for i := 0; i < 105; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		resp, _ := app.Test(req, -1)
		lastStatus = resp.StatusCode
	}
	assert.Equal(t, 429, lastStatus, "expected 429 after exceeding 100 requests per IP")
}

func TestRateLimiter_FirstRequestSetsExpiry(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newRateLimiterApp(rdb)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Find the rate-key that was created (prefix "rate:").
	keys := mr.Keys()
	var found bool
	for _, k := range keys {
		if len(k) > 5 && k[:5] == "rate:" {
			ttl := mr.TTL(k)
			assert.Positive(t, ttl, fmt.Sprintf("expected TTL to be set on key %q", k))
			found = true
			break
		}
	}
	assert.True(t, found, "expected a rate: key to exist in redis after the first request")
}

func TestRateLimiter_RedisErrorPassesThrough(t *testing.T) {
	// Start miniredis, capture the address, then close it so all ops fail.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr() // capture addr before closing
	mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: 0})
	app := newRateLimiterApp(rdb)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, _ := app.Test(req, -1)
	// When redis is unavailable the middleware should fail-open (pass through).
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

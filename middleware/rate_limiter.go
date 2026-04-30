package middleware

import (
	"fmt"
	"time"

	"url-shortener/config"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var key string
		var limit int64

		userID := c.Locals("userID")

		if userID != nil {
			key = "rate:user:" + fmt.Sprintf("%v", userID)
			limit = 1000 // logged-in users
		} else {
			ip := c.IP()
			key = "rate:ip:" + ip
			limit = 100 // guests
		}

		count, err := rdb.Incr(config.Ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(config.Ctx, key, time.Hour)
		}

		if count > limit {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
			})
		}

		return c.Next()
	}
}

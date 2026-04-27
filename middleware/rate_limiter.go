package middleware

import (
	"time"

	"url-shortener/config"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {

		ip := c.IP()
		key := "rate:" + ip

		count, err := rdb.Incr(config.Ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(config.Ctx, key, time.Hour)
		}

		if count > 10 {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
			})
		}

		return c.Next()
	}
}

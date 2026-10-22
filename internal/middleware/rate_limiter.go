package middleware

import (
	"time"

	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func SetupRateLimiter(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "Too many requests. Please slow down.", nil)
		},
	})
}

// SetupAuthRateLimiter dedicated to sensitive auth endpoints (anti brute-force)
func SetupAuthRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        10, // 10 attempts per minute
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "Too many login attempts. Please try again after 1 minute.", nil)
		},
	})
}

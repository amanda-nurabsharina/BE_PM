package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupCORS(allowedOrigins string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			// If allowedOrigins is empty or "*", allow all origins
			if allowedOrigins == "" || allowedOrigins == "*" {
				return true
			}

			// Check exact match from comma-separated list
			for _, allowed := range strings.Split(allowedOrigins, ",") {
				trimmed := strings.TrimSpace(allowed)
				if trimmed == "*" || strings.EqualFold(trimmed, origin) {
					return true
				}
			}

			// Fallback: automatically allow localhost or fourplusone.my.id subdomains
			if strings.HasSuffix(origin, ".fourplusone.my.id") || strings.Contains(origin, "localhost") {
				return true
			}

			return false
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
	})
}


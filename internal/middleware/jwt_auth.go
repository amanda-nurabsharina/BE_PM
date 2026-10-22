package middleware

import (
	"strings"

	"be-pm/pkg/response"
	"be-pm/pkg/security"

	"github.com/gofiber/fiber/v2"
)

func RequireJWT(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Error(c, fiber.StatusUnauthorized, "Missing Authorization header", nil)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid Authorization header format. Expected 'Bearer <token>'", nil)
		}

		claims, err := security.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid or expired access token", err.Error())
		}

		c.Locals("user", claims)
		return c.Next()
	}
}

func GetAuthUser(c *fiber.Ctx) *security.JWTClaims {
	u, ok := c.Locals("user").(*security.JWTClaims)
	if !ok {
		return nil
	}
	return u
}

package middleware

import (
	"net/http"
	"strings"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func Jwt(secret string, allowedRoles ...string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SuccessHandler: successHandler(allowedRoles),
		TokenLookup:    "header:Authorization",
		AuthScheme:     "Bearer",
		SigningKey:     jwtware.SigningKey{Key: []byte(secret)},
		ErrorHandler: func(c *fiber.Ctx, _ error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		},
	})
}

func successHandler(allowedRoles []string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		}
		rawUserId, ok := claims["user_id"].(string)
		if !ok || strings.TrimSpace(rawUserId) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		}
		rawRole, ok := claims["role"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		}

		c.Locals("user_id", rawUserId)
		c.Locals("role", rawRole)

		if len(allowed) == 0 {
			return c.Next()
		}
		if _, exists := allowed[rawRole]; !exists {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_REQUEST",
					"message": "invalid request",
				},
			})
		}

		return c.Next()
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/ghaniswara/dating-app/pkg/jwt"
	"github.com/labstack/echo"
)

func JWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing token"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token format"})
			}
			token := parts[1]

			_, err := jwt.ValidateToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token"})
			}

			return next(c)
		}
	}
}

package middleware

import (
	"net/http"

	"github.com/ghaniswara/dating-app/pkg/jwt"
	"github.com/labstack/echo"
)

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get("Authorization")
		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing token"})
		}

		claims, err := jwt.ValidateToken(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token"})
		}

		// Set context value
		c.Set("claims", claims)

		return next(c)
	}
}

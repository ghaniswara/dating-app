package middleware

import (
	"net/http"
	"strings"

	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	"github.com/ghaniswara/dating-app/pkg/jwt"
	"github.com/labstack/echo"
)

func JWTMiddleware(userRepo userRepo.IUserRepo) echo.MiddlewareFunc {
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

			claims, err := jwt.ValidateToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token"})
			}

			userProfile, err := userRepo.GetUserByID(c.Request().Context(), claims.UserID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token"})
			}

			// Set context value
			c.Set("claims", claims)
			c.Set("userProfile", userProfile)

			return next(c)
		}
	}
}

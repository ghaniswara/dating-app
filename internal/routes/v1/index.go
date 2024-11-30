package routesV1

import (
	routesV1Auth "github.com/ghaniswara/dating-app/internal/routes/v1/auth"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	"github.com/labstack/echo"
)

func InitV1Routes(e *echo.Echo, authCase *authUseCase.IAuthUseCase) {
	v1 := e.Group("/v1")

	v1.POST("/auth/sign-up", func(c echo.Context) error {
		return routesV1Auth.SignUpHandler(c, authCase)
	})
	v1.POST("/auth/sign-in", func(c echo.Context) error {
		return routesV1Auth.SignInHandler(c, authCase)
	})
}

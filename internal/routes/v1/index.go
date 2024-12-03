package routesV1

import (
	"github.com/ghaniswara/dating-app/internal/middleware"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	routesV1Auth "github.com/ghaniswara/dating-app/internal/routes/v1/auth"
	routesV1Match "github.com/ghaniswara/dating-app/internal/routes/v1/match"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	matchUseCase "github.com/ghaniswara/dating-app/internal/usecase/match"
	"github.com/labstack/echo"
)

func InitV1Routes(
	e *echo.Echo,
	authCase authUseCase.IAuthUseCase,
	matchCase matchUseCase.IMatchUseCase,
	userRepo userRepo.IUserRepo,
) {
	v1 := e.Group("/v1")

	authGroup := v1.Group("/auth")
	authGroup.POST("/sign-up", func(c echo.Context) error {
		return routesV1Auth.SignUpHandler(c, authCase)
	})
	authGroup.POST("/sign-in", func(c echo.Context) error {
		return routesV1Auth.SignInHandler(c, authCase)
	})

	matchGroup := v1.Group("/match", middleware.JWTMiddleware(userRepo))
	matchGroup.GET("/profile", func(c echo.Context) error {
		return routesV1Match.GetProfileHandler(c, matchCase)
	})
	matchGroup.POST("/profile/:id/like", func(c echo.Context) error {
		return routesV1Match.LikeHandler(c, matchCase)
	})

	matchGroup.POST("/profile/:id/pass", func(c echo.Context) error {
		return routesV1Match.PassHandler(c, matchCase)
	})
}

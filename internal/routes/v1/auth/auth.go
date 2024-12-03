package routesV1Auth

import (
	"net/http"

	"github.com/ghaniswara/dating-app/internal/entity"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	serializer "github.com/ghaniswara/dating-app/pkg/http_util"
	"github.com/labstack/echo"
)

func SignUpHandler(c echo.Context, authCase authUseCase.IAuthUseCase) error {
	reqBody, err := serializer.Decode[entity.CreateUserRequest](c)

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	problems := reqBody.Validate(c.Request().Context())

	if len(problems) != 0 {
		return serializer.Encode(c, 400, serializer.JSONResponse{
			Message: "Bad request check your request",
		})
	}

	_, err = (authCase).SignupUser(c.Request().Context(), reqBody)

	if err != nil {
		return serializer.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to sign up"})
	}

	return serializer.Encode(c, http.StatusOK, map[string]string{"message": "Sign-up successful"})
}

func SignInHandler(c echo.Context, authCase authUseCase.IAuthUseCase) error {
	reqBody, err := serializer.Decode[entity.SignInRequest](c)

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	problems := reqBody.Validate(c.Request().Context())

	if len(problems) != 0 {
		return serializer.Encode(c, 400, serializer.JSONResponse{
			Message: "Bad request check your request",
		})
	}

	jwtToken, err := (authCase).SignIn(c.Request().Context(), reqBody.Email, reqBody.Username, reqBody.Password)

	if err != nil {
		return serializer.Encode(c, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	return serializer.Encode(c, http.StatusOK, map[string]string{"token": jwtToken})
}

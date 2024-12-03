package routesV1Auth

import (
	"net/http"

	"github.com/ghaniswara/dating-app/internal/entity"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	"github.com/ghaniswara/dating-app/pkg/http_util"

	"github.com/labstack/echo"
)

func SignUpHandler(c echo.Context, authCase authUseCase.IAuthUseCase) error {
	reqBody, err := http_util.Decode[entity.CreateUserRequest](c)

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	problems := reqBody.Validate(c.Request().Context())

	if len(problems) != 0 {
		return http_util.Encode(c, 400, http_util.JSONResponse{
			Message: "Bad request check your request",
		})
	}

	user, err := (authCase).SignupUser(c.Request().Context(), reqBody)

	if err != nil {
		return http_util.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to sign up"})
	}

	response := entity.SignUpResponse{
		ID:       int(user.ID),
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
	}

	return http_util.Encode(c, http.StatusOK, http_util.HTTPResponse[entity.SignUpResponse]{
		Message: "Sign-up successful",
		Data:    response,
	})
}

func SignInHandler(c echo.Context, authCase authUseCase.IAuthUseCase) error {
	reqBody, err := http_util.Decode[entity.SignInRequest](c)

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	problems := reqBody.Validate(c.Request().Context())

	if len(problems) != 0 {
		return http_util.Encode(c, 400, http_util.JSONResponse{
			Message: "Bad request check your request",
		})
	}

	jwtToken, err := authCase.SignIn(c.Request().Context(), reqBody.Email, reqBody.Username, reqBody.Password)

	if err != nil {
		return http_util.Encode(c, http.StatusUnauthorized, http_util.HTTPErrorResponse[entity.SignInResponse]{
			Errors: []http_util.ErrorResponse{{Property: "request", Detail: "invalid credentials"}},
		})
	}

	return http_util.Encode(c, http.StatusOK, http_util.HTTPResponse[entity.SignInResponse]{
		Message: "Sign-in successful",
		Data:    entity.SignInResponse{Token: jwtToken},
	})
}

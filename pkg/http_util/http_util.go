package http_util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

type JSONResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Property string `json:"property"`
	Detail   string `json:"detail"`
}

type Validate interface {
	Validate(ctx context.Context) []ErrorResponse
}

func Encode[T any](c echo.Context, status int, v T) error {
	return c.JSON(status, v)
}

func Decode[T any](c echo.Context) (T, error) {
	var v T
	if err := c.Bind(&v); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResponse[T]{
			HTTPResponse: HTTPResponse[T]{
				Message: "Bad Request",
			},
			Errors: []ErrorResponse{{Property: "request", Detail: "check your request"}},
		})
		return v, err
	}
	return v, nil
}

func DecodeBody[T any](body []byte, v T) (T, error) {
	if err := json.Unmarshal(body, &v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	if _, ok := any(v).(T); !ok {
		return v, fmt.Errorf("unmarshaled data is not of type %T", v)
	}
	return v, nil
}

func ValidateRequest[T Validate](c echo.Context) (v T, err error) {
	problems := v.Validate(c.Request().Context())

	if len(problems) > 0 {
		return v, c.JSON(400, HTTPErrorResponse[T]{
			HTTPResponse: HTTPResponse[T]{
				Message: "Bad Request",
			},
			Errors: problems,
		})
	}
	return v, nil
}

type HTTPResponse[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type HTTPErrorResponse[T any] struct {
	HTTPResponse[T]
	Errors []ErrorResponse `json:"errors"`
}

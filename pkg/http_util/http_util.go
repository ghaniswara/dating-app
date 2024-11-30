package serializer

import (
	"context"
	"fmt"

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
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func ValidateRequest[T Validate](c echo.Context) (v T, err error) {
	problems := v.Validate(c.Request().Context())

	if len(problems) > 0 {
		// Create an error response
		errorResponse := ErrorResponse{
			Property: problems[0].Property, // Assuming you want to return the first problem
			Detail:   problems[0].Detail,
		}
		// Return the error response with a 400 status
		return v, c.JSON(400, errorResponse)
	}
	return v, nil
}

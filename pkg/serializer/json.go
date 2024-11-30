package serializer

import (
	"fmt"

	"github.com/labstack/echo"
)

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

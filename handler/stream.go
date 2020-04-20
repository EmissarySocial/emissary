package handler

import (
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

func GetStream(factory service.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// domain := factory.Domain()
		// stream := factory.Stream()

		return nil
	}
}

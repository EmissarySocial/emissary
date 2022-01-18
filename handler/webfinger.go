package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetWebfinger returns public webfinger information for a designated user
func GetWebfinger(maker *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		/*
			factory := maker.Factory(ctx.Request().Context())

			actor := factory.Actor()

			digit.Resource({}
		*/
		return nil
	}
}

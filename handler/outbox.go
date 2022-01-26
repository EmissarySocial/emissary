package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetOutbox returns an inbox for a particular ACTOR
func GetOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostOutbox accepts messages to a particular ACTOR
func PostOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

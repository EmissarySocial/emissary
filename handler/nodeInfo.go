package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetNodeInfo returns public webfinger information for a designated user
func GetNodeInfo(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

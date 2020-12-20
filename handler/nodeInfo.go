package handler

import (
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// GetNodeInfo returns public webfinger information for a designated user
func GetNodeInfo(factory *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

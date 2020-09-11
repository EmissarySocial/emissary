package handler

import (
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetNodeInfo returns public webfinger information for a designated user
func GetNodeInfo(factory *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

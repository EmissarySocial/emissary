package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

func StartupWizard(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		// TODO: Verify that the domain has not been initialized.
		// If it has, then forward to the setup page.
		return next
	}
}

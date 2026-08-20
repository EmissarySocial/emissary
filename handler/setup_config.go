package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

// SetupGetConfig serves the whole server configuration as JSON, for the setup console
func SetupGetConfig(factory *server.SetupFactory) func(c echo.Context) error {

	return func(c echo.Context) error {

		// Get the configuration from the factory
		result := mapof.Any{
			"config":    factory.Config(),
			"templates": factory.Template().Names(),
			"emails":    factory.Email().Names(),
		}

		// Return the configuration as JSON
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

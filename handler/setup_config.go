package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

func SetupGetConfig(factory *server.Factory) func(c echo.Context) error {

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

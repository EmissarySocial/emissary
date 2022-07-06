package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// GetOEmbed will provide an OEmbed service to be used exclusively by websites on this domain.
func GetOEmbed(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handlers.GetOEmbed", "Can't get domain")
		}

		return ctx.JSON(200, factory.Hostname())
	}
}

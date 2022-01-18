package handler

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetOEmbed will provide an OEmbed service to be used exclusively by websites on this domain.
func GetOEmbed(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "whisper.handlers.GetOEmbed", "Can't get domain")
		}

		return ctx.JSON(200, factory.Hostname())
	}
}

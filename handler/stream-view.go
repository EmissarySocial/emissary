package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

///////////////////////////////////////////////////////
// REQUEST HANDLERS

// GetStream returns an echo.HandlerFunc that displays a transition form
func GetStream(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Load the stream
		factory, stream, err := loadStream(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error Loading Stream"))
		}

		// Get the renderer
		renderer := factory.StreamViewer(ctx, stream, ctx.Param("view"))

		// Render the draft stream
		return derp.Report(renderStream(ctx, factory, renderer))
	}
}

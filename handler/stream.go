package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetStream generates the base HTML for a stream
func GetStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get the service factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Unrecognized domain"))
		}

		// Get the stream service
		streamService := factory.Stream()
		stream, err := streamService.LoadByToken(ctx.Param("token"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		// Render inner content
		result, err := factory.StreamRenderer(stream, getStreamLayout(ctx), getStreamView(ctx)).Render()

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering innerHTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func getStreamLayout(ctx echo.Context) string {

	if ctx.Request().Header.Get("hx-request") == "true" {
		return "stream-partial"
	}

	return "stream-full"
}

func getStreamView(ctx echo.Context) string {

	if view := ctx.QueryParam("view"); view != "" {
		return view
	}

	return "default"
}

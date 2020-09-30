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

		var result string

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

		// Render page content (full or partial)
		renderer := factory.StreamRenderer(*stream, getView(ctx))
		result, err = renderPage(factory.Layout(), renderer, isFullPageRequest(ctx))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering HTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func getView(ctx echo.Context) string {

	if view := ctx.QueryParam("view"); view != "" {
		return view
	}

	return "default"
}

func isFullPageRequest(ctx echo.Context) bool {
	return (ctx.Request().Header.Get("hx-request") != "true")
}

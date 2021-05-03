package handler

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// GetTemplates is a shortcut for the `GetLayout` handler, that only returns the
// template for creating new streams.
func GetTemplates(factoryManager *server.FactoryManager) echo.HandlerFunc {

	handler := GetLayout(factoryManager)

	return func(ctx echo.Context) error {
		ctx.SetParamNames("stream", "file")
		ctx.SetParamValues(ctx.Param("stream"), "stream-new-template")
		return handler(ctx)
	}
}

// GetLayout returns an echo.HandlerFunc that renders a specific site-wide layout with the given stream
func GetLayout(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		var result bytes.Buffer

		factory, stream, err := loadStream(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		// Get the renderer
		renderer := factory.StreamViewer(ctx, *stream, "default")

		// Render full page (stream only).
		layoutService := factory.Layout()
		template := layoutService.Template
		layoutFile := ctx.Param("file")

		if err := template.ExecuteTemplate(&result, layoutFile, renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
		}

		return ctx.HTML(200, result.String())
	}
}

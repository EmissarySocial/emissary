package handler

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetLayout returns an echo.HandlerFunc that renders a specific site-wide layout with the given stream
func GetLayout(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var result bytes.Buffer
		var stream model.Stream

		sterankoContext := ctx.(*steranko.Context)

		// Get the factory based on context Domain information
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetLayout", "Unrecognized Domain")
		}

		// Try to load the stream from the database
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.GetLayout", "Error loading stream", streamToken)
		}

		// Try to make a renderer.  This also includes permissions...
		renderer, err := render.NewRenderer(factory, sterankoContext, stream, "default")

		// Render template from the Layout
		layoutService := factory.Layout()
		template := layoutService.Template
		layoutFile := ctx.Param("file")

		if err := template.ExecuteTemplate(&result, layoutFile, renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
		}

		return ctx.HTML(200, result.String())
	}
}

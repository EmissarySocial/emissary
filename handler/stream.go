package handler

import (
	"bytes"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetStream handles GET requests
func GetStream(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var stream model.Stream

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetStream", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.GetStream", "Error loading Stream", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := factory.Renderer(sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetStream", "Error creating Renderer")
		}

		// Partial Page requests are simpler.
		if renderer.IsPartialRequest() {
			result, err := renderer.Render()

			if err != nil {
				return derp.Wrap(err, "ghost.handler.GetStream", "Error rendering stream")
			}
			return ctx.HTML(http.StatusOK, string(result))
		}

		// Full Page requests require the layout service
		layoutService := factory.Layout()
		var buffer bytes.Buffer

		if err := layoutService.Template.ExecuteTemplate(&buffer, "page", &renderer); err != nil {
			return derp.Wrap(err, "ghost.renderer.GetStream", "Error rendering full-page content")
		}

		return ctx.HTML(http.StatusOK, buffer.String())
	}
}

// PostStream handles POST/DELETE requests
func PostStream(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var stream model.Stream

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostStream", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.PostStream", "Error loading Stream", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := factory.Renderer(sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostStream", "Error creating Renderer")
		}

		action := renderer.Action()

		if err := render.DoPipeline(&renderer, ctx.Response().Writer, action.Steps, render.ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.renderer.PostStream", "Error executing action")
		}

		return nil
	}
}

// getStreamToken returns the :stream token from the Request (or a default)
func getStreamToken(ctx echo.Context) string {
	if token := ctx.Param("stream"); token != "" {
		return token
	}

	return "home"
}

// getActionID returns the :action token from the Request (or a default)
func getActionID(ctx echo.Context) string {

	if ctx.Request().Method == http.MethodDelete {
		return "delete"
	}

	if actionID := ctx.Param("action"); actionID != "" {
		return actionID
	}

	return "view"
}

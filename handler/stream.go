package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
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
		renderer, err := factory.Renderer(sterankoContext, stream, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostStream", "Error creating Renderer")
		}

		action := renderer.Action()

		// Execute all of the steps of the requested action
		for _, stepInfo := range action.Steps {

			step, err := factory.RenderStep(action.ActionID, stepInfo)

			if err != nil {
				return derp.Wrap(err, "ghost.renderer.PostStream", "Error initializing command", stepInfo)
			}

			if err := step.Get(&renderer); err != nil {
				return derp.Wrap(err, "ghost.renderer.PostStream", "Error executing command", stepInfo)
			}
		}

		return nil
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
		renderer, err := factory.Renderer(sterankoContext, stream, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostStream", "Error creating Renderer")
		}

		action := renderer.Action()

		// Execute all of the steps of the requested action
		for _, stepInfo := range action.Steps {

			step, err := factory.RenderStep(actionID, stepInfo)

			if err != nil {
				return derp.Wrap(err, "ghost.renderer.PostStream", "Error initializing command", stepInfo)
			}

			if err := step.Post(&renderer); err != nil {
				return derp.Wrap(err, "ghost.renderer.PostStream", "Error executing command", stepInfo)
			}
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

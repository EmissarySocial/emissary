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
			return derp.Wrap(err, "ghost.handler.StreamGet", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.StreamGet", "Error loading Stream", streamToken)
		}

		// Cast the context to a sterankoContext (so we can access the underlying Authorization)
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := render.NewRenderer(factory, sterankoContext, stream, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StreamGet", "Error creating renderer")
		}

		// If this is a partial page request, then we only need to render the stream content.
		if isPartialPageRequest(ctx) {
			result, err := renderer.Render()

			if err != nil {
				return derp.Wrap(err, "ghost.handler.StreamGet", "Error rendering content")
			}

			return ctx.HTML(http.StatusOK, string(result))
		}

		// Fall through means we're rendering this with the full page layout template
		var result bytes.Buffer
		layoutService := factory.Layout()
		template := layoutService.Template

		if err := template.ExecuteTemplate(&result, "page", renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
		}

		return ctx.HTML(200, result.String())
	}

}

// PostStream handles POST/DELETE requests
func PostStream(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var stream model.Stream

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StreamPost", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamToken := getStreamToken(ctx)
		actionID := getActionID(ctx)
		streamService := factory.Stream()

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.StreamPost", "Error loading Stream", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(steranko.Context)
		authorization := getAuthorization(&sterankoContext)
		action, err := render.NewAction(factory, &stream, &authorization, actionID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StreamPost", "Error finding actionConfig", stream, actionID)
		}

		// Almost done.  Let the action finish the request and
		// determine what response/headers to send back to the client.
		return action.Post(sterankoContext, &stream)
	}
}

// isPartialPageRequest returns TRUE if this request was made by `hx-get`
func isPartialPageRequest(ctx echo.Context) bool {
	return (ctx.Request().Header.Get("HX-Request") != "")
}

// getAuthorization unwraps the model.Authorization object that is embedded in the context.
func getAuthorization(sterankoContext *steranko.Context) model.Authorization {

	// get the authorization from the steranko.Context.  The context can ONLY be this one type.
	authorization, err := sterankoContext.Authorization()

	// handle errors
	if err != nil {
		return model.Authorization{}
	}

	// Cast the result as a model.Authorization object.  The authorization can ONLY be this one type.
	return authorization.(model.Authorization)
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

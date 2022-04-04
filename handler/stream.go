package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

// GetStream handles GET requests
func GetStream(factoryManager *server.Factory) echo.HandlerFunc {
	return renderStream(factoryManager, render.ActionMethodGet)
}

func PostStream(factoryManager *server.Factory) echo.HandlerFunc {
	return renderStream(factoryManager, render.ActionMethodPost)
}

// renderStream is the common Stream handler for both GET and POST requests
func renderStream(factoryManager *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderStream"

	return func(ctx echo.Context) error {

		stream := model.NewStream()

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {

			// Special case: If the HOME page is missing, then this is a new database.  Forward to the admin section
			if streamToken == "home" {
				return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
			}

			return derp.Wrap(err, location, "Error loading Stream by Token", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating Renderer")
		}

		if err := renderPage(factory, sterankoContext, &renderer, actionMethod); err != nil {
			return derp.Wrap(err, location, "Error rendering page")
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

package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// PostForm returns an echo.HandlerFunc that accepts form posts
// and performs actions on streams based on the user's permissions.
func PostForm(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get form data
		form := make(map[string]interface{})

		if err := ctx.Bind(&form); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load parse form data"))
		}

		// Get Factory and services required for this step
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error getting domain"))
		}

		// Load the current stream
		streamService := factory.Stream()
		stream, err := streamService.LoadByToken(ctx.Param("stream"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load stream"))
		}

		// Execute Transition
		transition, err := streamService.Transition(stream, ctx.Param("transitionId"), form)
		
		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
		}

		/// Render the stream
		result, err := factory.StreamRenderer(stream).Render(transition.NextView)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering innerHTML"))
		}

		// Success!
		return ctx.HTML(http.StatusOK, string(result))
	}
}

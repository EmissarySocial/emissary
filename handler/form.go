package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetForm generates an HTML form for the requested stream/transition
func GetForm(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get factory for this context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetForm", "Unrecognized domain")
		}

		// Try to load required values
		streamService := factory.Stream()
		token := ctx.Param("stream")
		transitionID := ctx.Param("transitionId")
		stream, err := streamService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Cannot load Stream", token))
		}

		// Render the HTML
		// Render page content (full or partial)
		renderer := factory.FormRenderer(*stream, transitionID)
		result, err := renderPage(factory.Layout(), renderer, isFullPageRequest(ctx))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering HTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

// PostForm returns an echo.HandlerFunc that accepts form posts
// and performs actions on streams based on the user's permissions.
func PostForm(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get Factory and services required for this step
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return err
		}
		// Get parameters from context
		token := ctx.Param("stream")
		transitionID := ctx.Param("transitionId")

		form := make(map[string]interface{})

		if err := ctx.Bind(&form); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load parse form data"))
		}

		streamService := factory.Stream()
		templateService := factory.Template()

		nextView := "default"

		// Load stream
		stream, err := streamService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load stream", token))
		}

		// Load template
		template, err := templateService.Load(stream.Template)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load template", stream))
		}

		// Execute transition
		if transition, err := template.Transition(stream.State, transitionID); err == nil {

			if err := streamService.Transition(stream, template, transitionID, form); err != nil {
				return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
			}

			nextView = transition.NextView
		}

		/// RENDER THE STREAM HERE
		result, err := factory.StreamRenderer(*stream, nextView).Render()

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering innerHTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

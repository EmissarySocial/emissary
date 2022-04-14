package handler

import (
	"bytes"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/domain"
	"github.com/whisperverse/whisperverse/render"
)

// renderPage collects the logic to render complete vs. partial HTML pages.
func renderPage(factory *domain.Factory, ctx *steranko.Context, renderer render.Renderer, actionMethod render.ActionMethod) error {

	const location = "handler.renderPage"

	// If this is a POST, then execute the action pipeline
	if actionMethod == render.ActionMethodPost {

		pipeline := render.Pipeline(renderer.Action().Steps)

		if err := pipeline.Execute(factory, renderer, ctx.Response().Writer, actionMethod); err != nil {
			return derp.Wrap(err, "handler.PostAdmin", "Error executing action pipeline", pipeline)
		}
		return nil
	}

	// Partial Page requests are served directly from the renderer
	if renderer.IsPartialRequest() || !renderer.UseGlobalWrapper() {
		result, err := renderer.Render()

		if err != nil {
			return derp.Wrap(err, location, "Error rendering partial page request")
		}
		return ctx.HTML(http.StatusOK, string(result))
	}

	// Full Page requests require the layout service to wrap the rendered content
	htmlTemplate := factory.Layout().Global().HTMLTemplate
	var buffer bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&buffer, "page", renderer); err != nil {
		return derp.Wrap(err, location, "Error rendering full-page content")
	}

	return ctx.HTML(http.StatusOK, buffer.String())
}

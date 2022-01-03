package handler

import (
	"bytes"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/render"
	"github.com/benpate/steranko"
)

// renderPage collects the logic to render complete vs. partial HTML pages.
func renderPage(factory *domain.Factory, ctx *steranko.Context, renderer render.Renderer) error {

	// Partial Page requests are served directly from the renderer
	if renderer.IsPartialRequest() {
		result, err := renderer.Render()

		if err != nil {
			return derp.Wrap(err, "ghost.handler.renderPage", "Error rendering partial page request")
		}
		return ctx.HTML(http.StatusOK, string(result))
	}

	// Full Page requests require the layout service to wrap the rendered content
	htmlTemplate := factory.Layout().Global().HTMLTemplate
	var buffer bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&buffer, "page", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderPage", "Error rendering full-page content")
	}

	return ctx.HTML(http.StatusOK, buffer.String())
}

package handler

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/render"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// renderHTML collects the logic to render complete vs. partial HTML pages.
func renderHTML(factory *domain.Factory, ctx echo.Context, renderer render.Renderer, actionMethod render.ActionMethod) error {

	const location = "handler.renderHTML"
	var partialPage bytes.Buffer

	// Execute the action pipeline
	pipeline := render.Pipeline(renderer.Action().Steps)

	status := pipeline.Execute(factory, renderer, &partialPage, actionMethod)

	if status.Error != nil {
		return derp.Wrap(status.Error, location, "Error executing action pipeline", pipeline)
	}

	// Copy status values into the Response...
	status.Apply(ctx.Response())

	// Partial page requests can be completed here.
	if renderer.IsPartialRequest() || status.FullPage {
		return ctx.HTML(status.GetStatusCode(), partialPage.String())
	}

	// Full Page requests require the theme service to wrap the rendered content
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	renderer.SetContent(partialPage.String())
	var fullPage bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", renderer); err != nil {
		return derp.Wrap(err, location, "Error rendering full-page content")
	}

	return ctx.HTML(http.StatusOK, fullPage.String())
}

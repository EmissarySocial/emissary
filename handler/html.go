package handler

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// buildHTML collects the logic to build complete vs. partial HTML pages.
func buildHTML(factory *domain.Factory, ctx echo.Context, b builder.Builder, actionMethod builder.ActionMethod) error {

	const location = "handler.buildHTML"
	var partialPage bytes.Buffer

	// Execute the action pipeline
	pipeline := builder.Pipeline(b.Action().Steps)

	status := pipeline.Execute(factory, b, &partialPage, actionMethod)

	if status.Error != nil {
		return derp.Wrap(status.Error, location, "Error executing action pipeline")
	}

	// Copy status values into the Response...
	status.Apply(ctx.Response())

	// Partial page requests can be completed here.
	if b.IsPartialRequest() || status.FullPage {
		return ctx.HTML(status.GetStatusCode(), partialPage.String())
	}

	// Full Page requests require the theme service to wrap the built content
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	b.SetContent(partialPage.String())
	var fullPage bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", b); err != nil {
		return derp.Wrap(err, location, "Error building full-page content")
	}

	return ctx.HTML(http.StatusOK, fullPage.String())
}

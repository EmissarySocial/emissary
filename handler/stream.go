package handler

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/Masterminds/sprig/v3"
	"github.com/benpate/choose"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetStream generates the base HTML for a stream
func GetStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get the service factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Unrecognized domain"))
		}

		// Get the stream service
		streamService := factory.Stream()

		// Get the stream
		token := choose.String(ctx.Param("stream"), "home")
		stream, err := streamService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		return derp.Report(renderStream(ctx, factory, stream))
	}
}

// isFullPageRequest returns TRUE if this is a regular, full-page request (and FALSE if it is an HTMX partial page request)
func isFullPageRequest(ctx echo.Context) bool {
	return (ctx.Request().Header.Get("hx-request") != "true")
}

// renderStream does the work to generate HTML for a stream and send it to the requester
func renderStream(ctx echo.Context, factory *service.Factory, stream *model.Stream) error {

	var renderer *service.Renderer
	var compiledTemplate *template.Template
	var entryPoint string
	var result bytes.Buffer
	var err error

	layoutService := factory.Layout()

	// If there is a "transition" defined, then we're displaying a form
	if transition := ctx.Param("transition"); transition != "" {

		renderer = factory.FormRenderer(stream, transition)
		compiledTemplate = layoutService.Layout()
		entryPoint = "form"

	} else {

		// Otherwise, we only want to display the stream.

		// Build a StreamRenderer
		renderer = factory.StreamRenderer(stream)

		// Load the Template to display the stream
		template, err := factory.Template().Load(stream.Template)

		if err != nil {
			return derp.Wrap(err, "ghost.render.Stream.Render", "Unable to load stream template")
		}

		// Get the View inside of the Template
		view, err := template.View(stream.State, ctx.QueryParam("view"))
		
		if err != nil {
			return derp.Wrap(err, "ghost.render.Stream.Render", "Invalid view")
		}

		// Get the "pre-compiled" Template from the View
		compiledTemplate, err = view.Compiled()
		
		if err != nil {
			return derp.Wrap(err, "ghost.render.Stream.Render", "Error getting compiled template")
		}

		// By default, the entryPoint is the name of the view
		entryPoint = view.Name
	}

	// If this is a full-page request, then alias the current "compiledTemplate" into the layout
	// templates with the name "content".
	if isFullPageRequest(ctx) {

		layout := layoutService.Layout()

		// Get the page layout
		entryPoint = "page"
	
		// Combine the two parse trees.
		// TODO: Could this be done at load time, not for each page request?
		compiledTemplate, err = layout.AddParseTree("content", compiledTemplate.Tree)
	
		if err != nil {
			return derp.Wrap(err, "ghost.render.Stream.Render", "Unable to create parse tree")
		}
	}

	// Render the page using the entryPoint to identify the Golang Template.
	if err := compiledTemplate.Funcs(sprig.FuncMap()).ExecuteTemplate(&result, entryPoint, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.Stream.Render", "Error rendering partial page")
	}

	return ctx.HTML(http.StatusOK, result.String())
}
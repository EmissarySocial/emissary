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

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		return derp.Report(renderStream(ctx, factory, stream))
	}
}

// GetNewStream generates an HTML form where authenticated users can create a new stream
func GetNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := newStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error loading stream"))
		}

		return derp.Report(renderStream(ctx, factory, stream))
	}
}

// PostStream returns an echo.HandlerFunc that accepts form posts
// and performs actions on streams based on the user's permissions.
func PostStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStream", "Error Loading Stream"))
		}

		return derp.Report(postStream(ctx, factory, stream))
	}
}

// PostNewStream accepts POST requests and generates a new stream.
func PostNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := newStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStream", "Error Loading Stream"))
		}

		return derp.Report(postStream(ctx, factory, stream))
	}
}

func loadStream(ctx echo.Context, factoryManager *service.FactoryManager) (*service.Factory, *model.Stream, error) {

	// Get the service factory
	factory, err := factoryManager.ByContext(ctx)

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Unrecognized domain"))
	}

	// Get the stream service
	streamService := factory.Stream()

	// Get the stream
	token := choose.String(ctx.Param("stream"), "home")
	stream, err := streamService.LoadByToken(token)

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
	}

	return factory, stream, nil
}

func newStream(ctx echo.Context, factoryManager *service.FactoryManager) (*service.Factory, *model.Stream, error) {

	// Locate the domain we're working in
	factory, err := factoryManager.ByContext(ctx)
	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error locating domain"))
	}

	streamService := factory.Stream()
	stream, err := streamService.NewWithTemplate(ctx.QueryParam("parent"), ctx.QueryParam("template"))

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error creating new stream"))
	}

	return factory, stream, nil
}

// renderStream does the work to generate HTML for a stream and send it to the requester
func renderStream(ctx echo.Context, factory *service.Factory, stream *model.Stream) error {

	var renderer *service.Renderer
	var compiledTemplate *template.Template
	var entryPoint string
	var result bytes.Buffer
	var err error

	layoutService := factory.Layout()

	//spew.Dump("---")

	// If there is a "transition" defined, then we're displaying a form
	if transition := ctx.QueryParam("transition"); transition != "" {

		renderer = factory.StreamRenderer(stream, ctx.QueryParams())
		compiledTemplate = layoutService.Layout()
		entryPoint = "form"

		if isFullPageRequest(ctx) {
			entryPoint = "page"

			// TODO: alias "form" to "content"
			//compiledTemplate, err = layout.AddParseTree("content", compiledTemplate.Tree)

			if err != nil {
				return derp.Wrap(err, "ghost.handler.renderStream", "Unable to create parse tree")
			}
		}

	} else {

		// Otherwise, we only want to display the stream.

		// Build a StreamRenderer
		renderer = factory.StreamRenderer(stream, ctx.QueryParams())

		// Load the Template to display the stream
		template, err := factory.Template().Load(stream.Template)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Unable to load stream template")
		}

		// Get the View inside of the Template
		view, err := template.View(stream.State, ctx.QueryParam("view"))

		if err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Invalid view")
		}

		// Get the "pre-compiled" Template from the View
		compiledTemplate, err = view.Compiled()

		if err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error getting compiled template")
		}

		// By default, the entryPoint is the name of the view
		entryPoint = view.Name

		// spew.Dump(template.Label)
		// spew.Dump(view.Name)

		// Combine the two parse trees.
		// TODO: Could this be done at load time, not for each page request?
		layout := layoutService.Layout()

		// If this is a full-page request then the entry point is the page.
		if isFullPageRequest(ctx) {
			entryPoint = "page"

			compiledTemplate, err = layout.AddParseTree("content", compiledTemplate.Tree)

			if err != nil {
				return derp.Wrap(err, "ghost.handler.renderStream", "Unable to create parse tree")
			}
		} else {

			compiledTemplate, err = layout.AddParseTree(entryPoint, compiledTemplate.Tree)

			if err != nil {
				return derp.Wrap(err, "ghost.handler.renderStream", "Unable to create parse tree")
			}
		}
	}

	// spew.Dump(compiledTemplate.DefinedTemplates())

	// Render the page using the entryPoint to identify the Golang Template.
	if err := compiledTemplate.Funcs(sprig.FuncMap()).ExecuteTemplate(&result, entryPoint, renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering partial page")
	}

	return ctx.HTML(http.StatusOK, result.String())
}

func postStream(ctx echo.Context, factory *service.Factory, stream *model.Stream) error {

	// spew.Dump("--- postStream")
	// Parse and Bind form data first, so that we don't have to hit the database in cases where there's an error.
	form := make(map[string]interface{})

	if err := ctx.Bind(&form); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load parse form data"))
	}

	streamService := factory.Stream()

	// Execute Transition
	transition, err := streamService.Transition(stream, ctx.QueryParam("transition"), form)

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, "/"+stream.Token+"?view="+transition.NextState)
	//	return renderStream(ctx, factory, stream)
}

// isFullPageRequest returns TRUE if this is a regular, full-page request (and FALSE if it is an HTMX partial page request)
func isFullPageRequest(ctx echo.Context) bool {
	return (ctx.Request().Header.Get("hx-request") != "true")
}

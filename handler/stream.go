package handler

import (
	"bytes"

	"github.com/benpate/choose"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

///////////////////////////////////
// EXISTING STREAMS

// GetStream returns an echo.HandlerFunc that displays a transition form
func GetStream(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error Loading Stream"))
		}

		return derp.Report(renderStream(ctx, factory, stream))
	}
}

func PostStreamContent(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var content model.Content

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			spew.Dump(err)
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamContent", "Error Loading Stream"))
		}

		if err := ctx.Bind(&content); err != nil {
			spew.Dump(err)
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamContent", "Error binding data"))
		}

		if err := stream.SetContent(content); err != nil {
			spew.Dump(err)
			return derp.Report(err)
		}

		streamService := factory.Stream()

		if err := streamService.Save(stream, "edit content: "+content.Content); err != nil {
			spew.Dump(err)
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamContent", "Error saving stream"))
		}

		ctx.Response().Header().Add("HX-Redirect", "/"+stream.Token)
		return ctx.NoContent(200)
	}
}

///////////////////////////////////
// TRANSITIONS ON NEW STREAMS

// GetTemplates returns the "new template" page, allowing users to choose a new template to go underneath the current s
func GetTemplates(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return renderLayout(factoryManager, "stream-new-template")
}

// GetNewStreamFromTemplate generates an HTML form where authenticated users can create a new stream
func GetNewStreamFromTemplate(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := newStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error loading stream"))
		}

		return derp.Report(renderForm(ctx, factory, stream, "create"))
	}
}

// PostNewStreamFromTemplate accepts POST requests and generates a new stream.
func PostNewStreamFromTemplate(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := newStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStreamFromTemplate", "Error Loading Stream"))
		}

		return derp.Report(doTransition(ctx, factory, stream, "create"))
	}
}

///////////////////////////////////
// TRANSITIONS ON EXISTING STREAMS

// GetTransition returns an echo.HandlerFunc that displays a transition form
func GetTransition(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStream", "Error Loading Stream"))
		}

		return derp.Report(renderForm(ctx, factory, stream, ctx.Param("transition")))
	}
}

// PostTransition returns an echo.HandlerFunc that accepts form posts
// and performs actions on streams based on the user's permissions.
func PostTransition(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStream", "Error Loading Stream"))
		}

		transition := ctx.Param("transition")

		return derp.Report(doTransition(ctx, factory, stream, transition))
	}
}

// GetLayout returns an echo.HandlerFunc that renders a specific site-wide layout with the given stream
func GetLayout(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		var result bytes.Buffer

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		layoutService := factory.Layout()
		request := domain.NewHTTPRequest(ctx)
		renderer := factory.StreamRenderer(*stream, request)

		layoutFile := ctx.Param("file")

		// Render full page (stream only).
		template := layoutService.Template

		if err := template.ExecuteTemplate(&result, layoutFile, renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
		}

		return ctx.HTML(200, result.String())
	}
}

///////////////////////////////////
// UTILITY FUNCTIONS

// renderLayout returns an echo.HandlerFunc that renders a specific site-wide layout with the given stream
func renderLayout(factoryManager *server.FactoryManager, templateID string) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		var result bytes.Buffer

		factory, stream, err := loadStream(ctx, factoryManager)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		layoutService := factory.Layout()
		request := domain.NewHTTPRequest(ctx)
		renderer := factory.StreamRenderer(*stream, request)

		// Render full page (stream only).
		template := layoutService.Template

		if err := template.ExecuteTemplate(&result, templateID, renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
		}

		return ctx.HTML(200, result.String())
	}
}

// newStream generates a new stream in the domain hierarchy
func newStream(ctx echo.Context, factoryManager *server.FactoryManager) (*domain.Factory, *model.Stream, error) {

	// Locate the domain we're working in
	factory, err := factoryManager.ByContext(ctx)
	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error locating domain"))
	}

	streamService := factory.Stream()
	stream, err := streamService.NewWithTemplate(ctx.Param("stream"), ctx.Param("template"))

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error creating new stream"))
	}

	return factory, stream, nil
}

// loadStream loads an existing stream from the domain hierarchy
func loadStream(ctx echo.Context, factoryManager *server.FactoryManager) (*domain.Factory, *model.Stream, error) {

	// Get the service factory
	factory, err := factoryManager.ByContext(ctx)

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.loadStream", "Unrecognized domain"))
	}

	// Get the stream service
	streamService := factory.Stream()

	// Get the stream
	token := choose.String(ctx.Param("stream"), "home")
	stream, err := streamService.LoadByToken(token)

	if err != nil {
		if !derp.NotFound(err) {
			return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.loadStream", "Error loading stream"))
		}
	}

	return factory, stream, nil
}

// renderStream does the work to generate HTML for a stream and send it to the requester
func renderStream(ctx echo.Context, factory *domain.Factory, stream *model.Stream) error {

	var result bytes.Buffer

	view := ctx.QueryParam("view")
	request := domain.NewHTTPRequest(ctx)
	renderer := factory.StreamViewer(*stream, request, view)

	// Partial page requests (stream only)
	if request.Partial() {

		if html, err := renderer.Render(); err == nil {
			return ctx.HTML(200, string(html))
		} else {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering partial HTML template")
		}
	}

	// Render full page (stream only).
	layoutService := factory.Layout()
	template := layoutService.Template

	if err := template.ExecuteTemplate(&result, "page", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
	}

	return ctx.HTML(200, result.String())
}

// renderForm does the work to generate HTML for a stream and send it to the requester
func renderForm(ctx echo.Context, factory *domain.Factory, stream *model.Stream, transitionID string) error {

	var result bytes.Buffer

	layoutService := factory.Layout()
	request := domain.NewHTTPRequest(ctx)
	renderer := factory.StreamTransitioner(*stream, request, transitionID)

	if !renderer.CanTransition(renderer.TransitionID()) {
		return derp.New(derp.CodeForbiddenError, "ghost.handler.stream.renderForm", "Forbidden")
	}

	template := layoutService.Template

	if err := template.ExecuteTemplate(&result, "form", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderForm", "Error rendering HTML form", stream, request)
	}

	return ctx.HTML(200, result.String())
}

// doTransition updates a stream with new data from a Form post and executes the requested transition.
func doTransition(ctx echo.Context, factory *domain.Factory, stream *model.Stream, transitionID string) error {

	// verify authorization
	request := domain.NewHTTPRequest(ctx)
	renderer := factory.StreamTransitioner(*stream, request, transitionID)

	if !renderer.CanTransition(renderer.TransitionID()) {
		return derp.New(derp.CodeForbiddenError, "ghost.handler.stream.renderForm", "Forbidden")
	}

	// Parse and Bind form data first, so that we don't have to hit the database in cases where there's an error.
	form := make(map[string]interface{})

	if err := ctx.Bind(&form); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load parse form data"))
	}

	streamService := factory.Stream()

	// Execute Transition
	transitionResult, err := streamService.DoTransition(stream, transitionID, form, request.Authorization())

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
	}

	ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+stream.Token+`?view=`+transitionResult.NextState+`"}}`)

	return ctx.NoContent(200)
}

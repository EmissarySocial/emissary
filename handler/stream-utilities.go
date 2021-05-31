package handler

import (
	"bytes"

	"github.com/benpate/choose"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

///////////////////////////////////////////////////////
// RENDERING FUNCTIONS

// renderStream does the work to generate HTML for a stream and send it to the requester
func renderStream(ctx echo.Context, factory *domain.Factory, renderer domain.Renderer) error {

	var result bytes.Buffer

	// Partial page requests (stream only)
	if renderer.Partial() {

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

	// Get the renderer
	renderer := factory.StreamTransitioner(ctx, *stream, transitionID)

	// Verify authorization
	if !renderer.CanTransition(renderer.TransitionID()) {
		return derp.New(derp.CodeForbiddenError, "ghost.handler.stream.renderForm", "Forbidden")
	}

	// Get the layout
	layoutService := factory.Layout()
	template := layoutService.Template

	// Render the layout template
	if err := template.ExecuteTemplate(&result, "form", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderForm", "Error rendering HTML form", stream)
	}

	return ctx.HTML(200, result.String())
}

///////////////////////////////////////////////////////
// UPDATE FUNCTIONS

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
func loadStream(factoryManager *server.FactoryManager, ctx echo.Context) (*domain.Factory, *model.Stream, error) {

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

// loadStreamTemplate returns the stream (and its corresponding template) requested in the context
func loadStreamTemplate(factoryManager *server.FactoryManager, ctx echo.Context) (*model.Template, *model.Stream, error) {

	factory, stream, err := loadStream(factoryManager, ctx)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.handler.loadStreamTemplate", "Unrecognized Domain")
	}

	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.handler.loadStreamTemplate", "Unrecognized Template")
	}

	return template, stream, nil
}

/*
func loadStreamAction(factoryManager *server.FactoryManager, ctx echo.Context) (*model.Stream, *model.Action, error) {

	template, stream, err := loadStreamTemplate(factoryManager, ctx)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.handler.loadStreamAction", "Error Loading Stream and Template")
	}

	action := template.Actions[ctx.Param("action")]


}
*/

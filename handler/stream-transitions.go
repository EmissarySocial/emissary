package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

///////////////////////////////////////////////////////
// RENDER FORMS

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

// GetTransition returns an echo.HandlerFunc that displays a transition form
func GetTransition(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStream", "Error Loading Stream"))
		}

		return derp.Report(renderForm(ctx, factory, stream, ctx.Param("transition")))
	}
}

///////////////////////////////////////////////////////
// EXECUTE TRANSITIONS

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

// PostTransition returns an echo.HandlerFunc that accepts form posts
// and performs actions on streams based on the user's permissions.
func PostTransition(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, stream, err := loadStream(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStream", "Error Loading Stream"))
		}

		transition := ctx.Param("transition")

		return derp.Report(doTransition(ctx, factory, stream, transition))
	}
}

///////////////////////////////////////////////////////
// UTILITIES

// doTransition updates a stream with new data from a Form post and executes the requested transition.
func doTransition(ctx echo.Context, factory *domain.Factory, stream *model.Stream, transitionID string) error {

	// verify authorization
	renderer := factory.StreamTransitioner(ctx, *stream, transitionID)

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
	transitionResult, err := streamService.DoTransition(stream, transitionID, form, renderer.Authorization())

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
	}

	ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+stream.Token+`?view=`+transitionResult.NextState+`"}}`)

	return ctx.NoContent(200)
}

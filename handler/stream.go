package handler

import (
	"net/http"

	"github.com/benpate/choose"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/action"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func HandleAction(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var stream model.Stream
		var action action.Action
		var err error

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.HandleAction", "Unrecognized Domain")
		}

		// Cast the context to a sterankoContext (so we can access the underlying Authorization)
		sterankoContext := ctx.(steranko.Context)

		// Try to get the user's authorization
		authorization, err := getAuthorization(sterankoContext)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.HandleAction", "Error getting authorization")
		}

		// Try to load the requested stream and action
		if err := getStreamAndAction(ctx, factory, &stream, action); err != nil {
			return derp.Wrap(err, "ghost.handler.HandleAction", "Error loading stream")
		}

		// Verify user's permissions
		if !action.UserCan(&stream, &authorization) {
			return derp.New(derp.CodeForbiddenError, "ghost.handler.HandleAction", "Unauthorized")
		}

		// Call the appropriate method (get or post)
		if ctx.Request().Method == http.MethodGet {
			err = action.Get(sterankoContext, factory, &stream)

		} else {
			err = action.Post(sterankoContext, factory, &stream)
		}

		// Handle errors
		if err != nil {
			return derp.Wrap(err, "ghost.handler.HandleAction", "Error executing action")
		}

		// Success
		return nil
	}
}

// getStreamAndAction locates the correct stream and action objects referenced by the current context
func getStreamAndAction(ctx echo.Context, factory *domain.Factory, stream *model.Stream, action action.Action) error {

	var streamID string
	var actionID string

	// Try to load the stream from the database
	streamService := factory.Stream()

	streamID = choose.String(ctx.Param("stream"), "home")

	if err := streamService.LoadByToken(streamID, stream); err != nil {
		return derp.Wrap(err, "ghost.handler.getStreamAndAction", "Error Loading Stream")
	}

	// Try to load the template used by this stream
	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.handler.getStreamAndAction", "Error Loading Template")
	}

	// Calculate the actionID
	if ctx.Request().Method == http.MethodDelete {
		actionID = "delete"
	} else {
		actionID = ctx.Param("action")
	}

	// Check for missing actions
	if _, ok := template.Actions[actionID]; !ok {
		return derp.New(derp.CodeInternalError, "ghost.handler.getStreamAndAction", "Invalid Action")
	}

	// Set the action
	action = template.Actions[actionID]

	// Silence means success.
	return nil
}

// getAuthorization unwraps the model.Authorization object that is embedded in the context.
func getAuthorization(ctx steranko.Context) (model.Authorization, error) {

	// get the authorization from the steranko.Context.  The context can ONLY be this one type.
	authorization, err := ctx.Authorization()

	// handle errors
	if err != nil {
		return model.Authorization{}, derp.Wrap(err, "ghost.handler.getAuthorization", "Error retrieving authorization from context")
	}

	// Cast the result as a model.Authorization object.  The authorization can ONLY be this one type.
	return authorization.(model.Authorization), nil
}

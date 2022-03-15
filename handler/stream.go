package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetStream handles GET requests
func GetStream(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		stream := model.NewStream()

		// Try to get the factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.GetStream", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {

			// Special case: If the HOME page is missing, then this is a new database.  Forward to the admin section
			if streamToken == "home" {
				return ctx.Redirect(http.StatusTemporaryRedirect, "/admin/startup/toplevel")
			}
			return derp.Wrap(err, "whisper.handler.GetStream", "Error loading Stream by Token", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.GetStream", "Error creating Renderer")
		}

		return renderPage(factory, sterankoContext, &renderer)
	}
}

// PostStream handles POST/DELETE requests
func PostStream(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		stream := model.NewStream()

		// Try to get the Factory from the context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.PostStream", "Unrecognized Domain")
		}

		// Try to load the stream using request data
		streamService := factory.Stream()
		streamToken := getStreamToken(ctx)

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, "whisper.handler.PostStream", "Error loading Stream", streamToken)
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		actionID := getActionID(ctx)
		renderer, err := render.NewStreamWithoutTemplate(factory, sterankoContext, &stream, actionID)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.PostStream", "Error creating Renderer")
		}

		// Execute the action pipeline
		if action := renderer.Action(); action != nil {
			if err := render.DoPipeline(&renderer, ctx.Response().Writer, action.Steps, render.ActionMethodPost); err != nil {
				return derp.Wrap(err, "whisper.renderer.PostStream", "Error executing action")
			}
		}

		// Woot!!
		return nil
	}
}

// getStreamToken returns the :stream token from the Request (or a default)
func getStreamToken(ctx echo.Context) string {
	if token := ctx.Param("stream"); token != "" {
		return token
	}

	return "home"
}

// getActionID returns the :action token from the Request (or a default)
func getActionID(ctx echo.Context) string {

	if ctx.Request().Method == http.MethodDelete {
		return "delete"
	}

	if actionID := ctx.Param("action"); actionID != "" {
		return actionID
	}

	return "view"
}

// getSignedInUserID returns the UserID for the current request.
// If the authorization is not valid or not present, then the error contains http.StatusUnauthorized
func getSignedInUserID(ctx echo.Context) (primitive.ObjectID, error) {

	const location = "whisperverse.handler.getSignedInUserID"

	sterankoContext, ok := ctx.(*steranko.Context)

	if !ok {
		return primitive.NilObjectID, derp.New(http.StatusUnauthorized, location, "Invalid Authorization")
	}

	authorization, err := sterankoContext.Authorization()

	if err != nil {
		err = derp.Wrap(err, location, "Invalid Authorization")
		derp.SetErrorCode(err, http.StatusUnauthorized)
		return primitive.NilObjectID, err
	}

	auth, ok := authorization.(*model.Authorization)

	if !ok {
		return primitive.NilObjectID, derp.New(http.StatusUnauthorized, location, "Invalid Authorization", authorization)
	}

	return auth.UserID, nil

}

// isOnwer returns TRUE if the JWT Claim is from a domain owner.
func isOwner(claims jwt.Claims, err error) bool {

	if err == nil {
		if claims.Valid() == nil {
			if authorization, ok := claims.(*model.Authorization); ok {
				return authorization.DomainOwner
			}
		}
	}

	return false
}

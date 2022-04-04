package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

// GetProfile handles GET requests
func GetProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, render.ActionMethodGet)
}

// PostProfile handles POST/DELETE requests
func PostProfile(factoryManager *server.Factory) echo.HandlerFunc {
	return renderProfile(factoryManager, render.ActionMethodPost)
}

// renderProfile is the common Profile handler for both GET and POST requests
func renderProfile(fm *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderProfile"

	return func(ctx echo.Context) error {

		// Try to locate the domain from the Context
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to load the user profile by userId or username
		service := factory.User()
		user := model.NewUser()

		userToken := ctx.Param("user")
		actionID := getActionID(ctx)

		if err := service.LoadByToken(userToken, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userToken)
		}

		sterankoContext := ctx.(*steranko.Context)
		profileLayout := factory.Layout().Profile()
		action := profileLayout.Action(actionID)

		renderer := render.NewProfile(factory, sterankoContext, profileLayout, action, &user)

		return renderPage(factory, sterankoContext, renderer, actionMethod)
	}
}

/*
func GetProfileInbox(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.GetProfileInbox"

	return func(ctx echo.Context) error {

		// Try to get the domain factory
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Domain")
		}

		// Try to find the signed in user
		userID, err := getSignedInUserID(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error reading Authorization")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userID)
		}

		// Render the user's Inbox
		ctx.SetParamNames("stream")
		ctx.SetParamValues(user.InboxID.Hex())

		return GetStream(fm)(ctx)
	}
}

// TODO: This could maybe be merged with GetProfileInbox, once we
// understand everything required.
func GetProfileOutbox(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.GetProfileOutbox"

	return func(ctx echo.Context) error {

		// Try to get the domain factory
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Domain")
		}

		// Try to find the signed in user
		userID, err := getSignedInUserID(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error reading Authorization")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userID)
		}

		// Render the user's Inbox
		ctx.SetParamNames("stream")
		ctx.SetParamValues(user.OutboxID.Hex())

		return GetStream(fm)(ctx)

	}
}
*/

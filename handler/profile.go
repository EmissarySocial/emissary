package handler

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
)

func GetProfile(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

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

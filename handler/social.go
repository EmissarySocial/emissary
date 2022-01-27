package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/activitystream/writer"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
)

// GetSocialProfile returns a person's ActivityPub Actor profile
func GetSocialProfile(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.GetSocialProfile"

	return func(ctx echo.Context) error {

		// Try to load the domain factory for this request
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()
		userID := ctx.Param("userId")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User")
		}

		// Generate a profile page
		profile := "https://" + factory.Hostname() + "/people/" + user.UserID.Hex()

		result := writer.Person(user.DisplayName, "en").
			ID(profile).
			Summary(user.Description, "en").
			Icon(user.AvatarURL).
			Property("inbox", profile+"/inbox").
			Property("outbox", profile+"/outbox").
			Property("following", profile+"/following").
			Property("followers", profile+"/followers").
			Property("liked", profile+"/liked").
			Property("preferredUsername", user.Username)

		return ctx.JSON(http.StatusOK, result)
	}
}

// GetSocialInbox returns an inbox for a particular ACTOR
func GetSocialInbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

// PostSocialInbox accepts messages to a particular ACTOR
func PostSocialInbox(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.PostSocialInbox"

	return func(ctx echo.Context) error {

		// Try to get the domain factory
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain")
		}

		// Try to load the user who owns the inbox
		userService := factory.User()
		user := model.NewUser()
		userID := ctx.Param("userId")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userID)
		}

		// TODO: Validate signatures here

		// Try to import the ActivityPub record
		body := make(map[string]interface{})
		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Error binding request body")
		}

		inboxService := factory.Inbox()
		if err := inboxService.Receive(&user, body); err != nil {
			return derp.Wrap(err, location, "Error processing ActivityPub message")
		}

		return ctx.NoContent(http.StatusNoContent)
	}
}

// GetSocialOutbox returns an inbox for a particular ACTOR
func GetSocialOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostSocialOutbox accepts messages to a particular ACTOR
func PostSocialOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetSocialFollowers accepts messages to a particular ACTOR
func GetSocialFollowers(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostSocialFollowers accepts messages to a particular ACTOR
func PostSocialFollowers(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetSocialFollowing accepts messages to a particular ACTOR
func GetSocialFollowing(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetSocialLiked accepts messages to a particular ACTOR
func GetSocialLiked(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

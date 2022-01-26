package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/activitystream/writer"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
)

// GetProfile returns a person's ActivityPub Actor profile
func GetProfile(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.GetProfile"

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

// GetInbox returns an inbox for a particular ACTOR
func GetInbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

// PostInbox accepts messages to a particular ACTOR
func PostInbox(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.PostInbox"

	return func(ctx echo.Context) error {

		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain")
		}

		inboxService := factory.Inbox()

		body := make(map[string]interface{})

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Error binding request body")
		}

		// TODO: Validate signatures here

		if err := inboxService.Receive(body); err != nil {
			return derp.Wrap(err, location, "Error processing ActivityPub message")
		}

		return ctx.NoContent(http.StatusNoContent)
	}
}

// GetOutbox returns an inbox for a particular ACTOR
func GetOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostOutbox accepts messages to a particular ACTOR
func PostOutbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetFollowers accepts messages to a particular ACTOR
func GetFollowers(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostFollowers accepts messages to a particular ACTOR
func PostFollowers(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetFollowing accepts messages to a particular ACTOR
func GetFollowing(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// GetLiked accepts messages to a particular ACTOR
func GetLiked(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

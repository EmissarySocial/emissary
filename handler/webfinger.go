package handler

import (
	"net/http"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
)

// GetWebfinger returns public webfinger information for a designated user.
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(fm *server.Factory) echo.HandlerFunc {

	location := "whispervers.handler.GetWebFinger"

	return func(ctx echo.Context) error {

		// Validate Domain and Get Factory Object
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		resource := ctx.QueryParam("resource")

		// Handle User Requests
		if strings.HasPrefix(resource, "acct:") {

			userService := factory.User()
			user := model.NewUser()
			username := strings.TrimPrefix(resource, "acct:")

			// Try to load the User from the database
			if err := userService.LoadByUsername(username, &user); err != nil {
				return derp.Wrap(err, location, "Error loading User", username)
			}

			// Profile URL for this user
			profile := "https://" + factory.Hostname() + "/people/" + user.UserID.Hex()

			// Generate a WebFinger response
			result := digit.NewResource(resource).
				Link(digit.RelationTypeSelf, "application/activity+json", profile).
				Link(digit.RelationTypeProfile, "text/html", profile)

			return ctx.JSON(http.StatusOK, result)
		}

		// TODO: Handle Page Requests
		return derp.NewBadRequestError(location, "Resource Type Not Implemented", resource)
	}
}

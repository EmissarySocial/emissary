package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetProfile(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetProfile"

	return func(context echo.Context) error {

		// Try to find the factory for this context
		factory, err := serverFactory.ByContext(context)

		if err != nil {
			return derp.Wrap(err, location, "Error creating server factory")
		}

		// Try to load the user
		userService := factory.User()
		user := model.NewUser()
		userID := context.Param("userId")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", userID)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "User not found")
		}

		// Return the user's profile in JSON-LD format
		context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
		return context.JSON(http.StatusOK, user.GetJSONLD())
	}
}

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

	return func(context echo.Context) error {

		// Try to find the factory for this context
		factory, err := serverFactory.ByContext(context)

		if err != nil {
			return derp.Wrap(err, "handler.ActivityPub_GetProfile", "Error creating server factory")
		}

		// Try to load the user
		userService := factory.User()
		user := model.NewUser()
		userID := context.Param("userId")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, "handler.ActivityPub_GetProfile", "Error loading user", userID)
		}

		// Return the user's profile in JSON-LD format
		context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
		return context.JSON(http.StatusOK, user.GetJSONLD())
	}
}

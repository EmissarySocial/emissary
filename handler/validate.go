package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetValidateSignupCode validates a User.Username for uniqueness/availability
func GetValidateSignupCode(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// This service can only validate the "secret" field
	if field := ctx.QueryParam("field"); field != "secret" {
		return ctx.JSON(http.StatusBadRequest, mapof.Any{
			"valid":   false,
			"message": "Invalid field",
		})
	}

	// Validate the secret code against the registration template
	domain := factory.Domain().Get()

	if ctx.QueryParam("value") != domain.RegistrationData.GetString("secret") {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   false,
			"message": "",
		})
	}

	// If the username is allowed, then return a success
	return ctx.JSON(http.StatusOK, mapof.Any{
		"valid":   true,
		"message": "",
	})
}

// GetValidateUsername validates a User.Username for uniqueness/availability
func GetValidateUsername(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// This service can only validate the "username" field
	if field := ctx.QueryParam("field"); field != "username" {
		return ctx.JSON(http.StatusBadRequest, mapof.Any{
			"valid":   false,
			"message": "Invalid field",
		})
	}

	// Collect variables
	userService := factory.User()
	authorization := getAuthorization(ctx)
	userID := authorization.UserID
	username := ctx.QueryParam("value")

	// If the username is not allowed, then return an error
	if err := userService.ValidateUsername(session, userID, username); err != nil {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   false,
			"message": derp.Message(err),
		})
	}

	// If the username is allowed, then return a success
	return ctx.JSON(http.StatusOK, mapof.Any{
		"valid":   true,
		"message": "",
	})
}

// GetValidateStreamToken validates a Stream.Token for uniqueness/availability
func GetValidateStreamToken(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	token := ctx.QueryParam("value")

	if len(token) < 3 {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   false,
			"message": "Token must be at least 3 characters",
		})
	}

	// This service can only validate the "token" field
	if field := ctx.QueryParam("field"); field != "token" {
		return ctx.JSON(http.StatusBadRequest, mapof.Any{
			"valid":   false,
			"message": "Invalid field",
		})
	}

	// Collect variables
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByToken(session, ctx.QueryParam("value"), &stream); err != nil {

		if derp.IsNotFound(err) {
			return ctx.JSON(http.StatusOK, mapof.Any{
				"valid":   true,
				"message": "",
			})
		}

		return derp.Wrap(err, "handler.GetValidateStreamToken", "Error loading stream by token")
	}

	// If there is no match, then the token is valid
	if stream.ID() == ctx.QueryParam("streamId") {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   true,
			"message": "",
		})
	}

	// Otherwise, the token is taken
	return ctx.JSON(http.StatusOK, mapof.Any{
		"valid":   false,
		"message": "This token is already in use by another stream",
	})
}

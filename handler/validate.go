package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

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

	/*
		// This service can only validate the "token" field
		if field := ctx.QueryParam("field"); field != "token" {
			return ctx.JSON(http.StatusBadRequest, mapof.Any{
				"valid":   false,
				"message": "Invalid field",
			})
		}
	*/

	return derp.NotImplemented("handler.GetValidateStreamToken")
}

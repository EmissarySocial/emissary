package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	if domain := factory.Domain().Get(); ctx.QueryParam("value") != domain.RegistrationData.GetString("secret") {
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

// GetValidateFoldername validates a Folder.Label for uniqueness within the User's own Folders
func GetValidateFoldername(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	// This service can only validate the "label" field
	if field := ctx.QueryParam("field"); field != "label" {
		return ctx.JSON(http.StatusBadRequest, mapof.Any{
			"valid":   false,
			"message": "Invalid field",
		})
	}

	// The Folder being edited is excluded from the search.  A missing or malformed
	// folderId leaves it zero, which excludes nothing -- correct for new Folders.
	folderID, _ := primitive.ObjectIDFromHex(ctx.QueryParam("folderId"))

	// If the label is not allowed, then return an error
	if err := factory.Folder().ValidateLabel(session, user.UserID, folderID, ctx.QueryParam("value")); err != nil {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   false,
			"message": derp.Message(err),
		})
	}

	// If the label is allowed, then return a success
	return ctx.JSON(http.StatusOK, mapof.Any{
		"valid":   true,
		"message": "",
	})
}

// GetValidateCirclename validates a Circle.Name for uniqueness within the User's own Circles
func GetValidateCirclename(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	// This service can only validate the "name" field
	if field := ctx.QueryParam("field"); field != "name" {
		return ctx.JSON(http.StatusBadRequest, mapof.Any{
			"valid":   false,
			"message": "Invalid field",
		})
	}

	// The Circle being edited is excluded from the search.  A missing or malformed
	// circleId leaves it zero, which excludes nothing -- correct for new Circles.
	circleID, _ := primitive.ObjectIDFromHex(ctx.QueryParam("circleId"))

	// If the name is not allowed, then return an error
	if err := factory.Circle().ValidateName(session, user.UserID, circleID, ctx.QueryParam("value")); err != nil {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   false,
			"message": derp.Message(err),
		})
	}

	// If the name is allowed, then return a success
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

	if err := streamService.LoadByToken(session, token, &stream); err != nil {

		if derp.IsNotFound(err) {
			return ctx.JSON(http.StatusOK, mapof.Any{
				"valid":   true,
				"message": "",
			})
		}

		return derp.Wrap(err, "handler.GetValidateStreamToken", "Loading stream by token")
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

// GetValidateGroupToken validates a Group.Token for uniqueness/availability
func GetValidateGroupToken(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetValidateGroupToken"

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
	groupService := factory.Group()
	group := model.NewGroup()

	if err := groupService.LoadByToken(session, token, &group); err != nil {

		if derp.IsNotFound(err) {
			return ctx.JSON(http.StatusOK, mapof.Any{
				"valid":   true,
				"message": "",
			})
		}

		return derp.Wrap(err, location, "Loading group by token")
	}

	// If the only match is the Group being edited, then the token is still valid
	if group.ID() == ctx.QueryParam("groupId") {
		return ctx.JSON(http.StatusOK, mapof.Any{
			"valid":   true,
			"message": "",
		})
	}

	// Otherwise, the token is taken
	return ctx.JSON(http.StatusOK, mapof.Any{
		"valid":   false,
		"message": "This token is already in use by another group",
	})
}

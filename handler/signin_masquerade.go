package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostMasquerade(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostMasquerade"

	if !isOwner(ctx.Authorization()) {
		return derp.ForbiddenError(location, "Unauthorized")
	}

	// Collect the userID from the Request
	token := ctx.QueryParam("userId")
	userID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid User ID", token)
	}

	// Load the requested User
	user := model.NewUser()
	userService := factory.User()
	if err := userService.LoadByID(session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Unable to load User", derp.WithCode(http.StatusBadRequest))
	}

	// Create a masquerade certificate for the requested User
	if err := factory.Steranko(session).SigninUser(ctx, &user); err != nil {
		return derp.Wrap(err, location, "Unable to create JWT certificate")
	}

	// Forward to the user's profile page
	return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
}

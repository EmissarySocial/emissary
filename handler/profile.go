package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetProfile handles GET requests
func GetProfile(serverFactory *server.Factory) echo.HandlerFunc {
	return renderProfile(serverFactory, render.ActionMethodGet)
}

// PostProfile handles POST/DELETE requests
func PostProfile(serverFactory *server.Factory) echo.HandlerFunc {
	return renderProfile(serverFactory, render.ActionMethodPost)
}

// renderProfile is the common Profile handler for both GET and POST requests
func renderProfile(serverFactory *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.renderProfile"

	return func(context echo.Context) error {

		// Cast the context into a steranko context (which includes authentication data)
		sterankoContext := context.(*steranko.Context)

		// Get the domain factory from the context
		factory, err := serverFactory.ByContext(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain factory")
		}

		// Get the UserID from the URL (could be "me")
		userID, err := profileUserID(sterankoContext)

		if err != nil {
			return derp.Wrap(err, location, "Error loading user ID")
		}

		// Try to load the user from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user", userID)
		}

		// Try to load the User's Outbox
		actionID := first.String(context.Param("action"), "view")
		renderer, err := render.NewProfile(factory, sterankoContext, &user, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer")
		}

		// Forward to the standard page renderer to complete the job
		return renderPage(factory, sterankoContext, renderer, actionMethod)
	}
}

func profileUserID(context *steranko.Context) (primitive.ObjectID, error) {

	userIDString := context.Param("userId")

	if objectID, err := primitive.ObjectIDFromHex(userIDString); err == nil {
		return objectID, nil
	}

	if userIDString == "me" {
		authorization := getAuthorization(context)

		if authorization.IsAuthenticated() {
			return authorization.UserID, nil
		}

		return primitive.NilObjectID, derp.NewUnauthorizedError("handler.profileUserID", "Cannot use 'me' when user is not authenticated")
	}

	return primitive.ObjectIDFromHex(userIDString)
}

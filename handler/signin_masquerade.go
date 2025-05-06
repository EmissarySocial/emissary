package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostMasquerade(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostMasquerade"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Authenticate the request is from a Domain Owner
		s := factory.Steranko()
		sterankoContext := ctx.(*steranko.Context)

		if !isOwner(sterankoContext.Authorization()) {
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
		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", derp.WithCode(http.StatusBadRequest))
		}

		// Create a masquerade certificate for the requested User
		certificate, err := s.CreateCertificate(ctx.Request(), &user)

		if err != nil {
			return derp.Wrap(err, location, "Error creating JWT certificate")
		}

		// Push the certificate and make a -backup cookie
		s.PushCookie(ctx, certificate)

		// Forward to the user's profile page
		return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
	}
}

package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// getActionID returns the :action token from the Request (or a default)
func getActionID(ctx echo.Context) string {

	if ctx.Request().Method == http.MethodDelete {
		return "delete"
	}

	if actionID := ctx.Param("action"); actionID != "" {
		return actionID
	}

	return "view"
}

// getSignedInUserID returns the UserID for the current request.
// If the authorization is not valid or not present, then the error contains http.StatusUnauthorized
func getSignedInUserID(ctx echo.Context) (primitive.ObjectID, error) {

	const location = "whisperverse.handler.getSignedInUserID"

	sterankoContext, ok := ctx.(*steranko.Context)

	if !ok {
		return primitive.NilObjectID, derp.New(http.StatusUnauthorized, location, "Invalid Authorization")
	}

	authorization, err := sterankoContext.Authorization()

	if err != nil {
		err = derp.Wrap(err, location, "Invalid Authorization")
		derp.SetErrorCode(err, http.StatusUnauthorized)
		return primitive.NilObjectID, err
	}

	auth, ok := authorization.(*model.Authorization)

	if !ok {
		return primitive.NilObjectID, derp.New(http.StatusUnauthorized, location, "Invalid Authorization", authorization)
	}

	return auth.UserID, nil

}

// isOnwer returns TRUE if the JWT Claim is from a domain owner.
func isOwner(claims jwt.Claims, err error) bool {

	if err == nil {
		if claims.Valid() == nil {
			if authorization, ok := claims.(*model.Authorization); ok {
				return authorization.DomainOwner
			}
		}
	}

	return false
}

package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
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

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return *auth
		}
	}

	return model.NewAuthorization()
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

/*
// getSignedInUserID returns the UserID for the current request.
// If the authorization is not valid or not present, then the error contains http.StatusUnauthorized
func getSignedInUserID(ctx echo.Context) (primitive.ObjectID, error) {

	const location = "handler.getSignedInUserID"

	sterankoContext, ok := ctx.(*steranko.Context)

	if !ok {
		return primitive.NilObjectID, derp.NewUnauthorizedError(, location, "Invalid Authorization")
	}

	authorization, err := sterankoContext.Authorization()

	if err != nil {
		err = derp.Wrap(err, location, "Invalid Authorization")
		derp.SetErrorCode(err, http.StatusUnauthorized)
		return primitive.NilObjectID, err
	}

	auth, ok := authorization.(*model.Authorization)

	if !ok {
		return primitive.NilObjectID, derp.NewUnauthorizedError(, location, "Invalid Authorization", authorization)
	}

	return auth.UserID, nil

}

*/

// cleanQueryParams returns a "clean" version of a url.Values structure.
// It truncates all slices into a single string.
func cleanQueryParams(values url.Values) mapof.String {
	result := make(mapof.String, len(values))
	for key, value := range values {

		if len(value) == 0 {
			result[key] = ""
		} else {
			result[key] = value[0]
		}
	}

	return result
}

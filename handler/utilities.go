package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
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

func isUserVisible(context *steranko.Context, user *model.User) bool {

	authorization := getAuthorization(context)

	// Domain owners can see everything
	if authorization.DomainOwner {
		return true
	}

	// Signed-in users can see themselves
	if authorization.UserID == user.UserID {
		return true
	}

	// Otherwise, access depends on the user's profile being public
	return user.IsPublic
}

// isOnwer returns TRUE if the JWT Claim is from a domain owner.
func isOwner(claims jwt.Claims, err error) bool {

	if err == nil {
		if authorization, ok := claims.(*model.Authorization); ok {
			return authorization.DomainOwner
		}
	}

	return false
}

// cleanQueryParams returns a "clean" version of a url.Values structure.
// It truncates all slices into a single string.
func cleanQueryParams(values url.Values) mapof.Any {
	result := make(mapof.Any, len(values))
	for key, value := range values {
		if len(value) > 0 {
			result[key] = value[0]
		}
	}

	return result
}

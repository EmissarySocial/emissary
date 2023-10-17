package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
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

// getActualDocument traverses "Create" and "Update" messages to get the actual document that we want to save
func getActualDocument(document streams.Document) streams.Document {

	// Load the full version of the document (if it's a link)
	loaded, err := document.Load()

	if err != nil {
		return document
	}

	switch loaded.Type() {

	// If the document is a "Create" activity, then we want to use the object as the actual message
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return loaded.Object()

	// Otherwise, we'll just use the document as-is
	default:
		return loaded
	}
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

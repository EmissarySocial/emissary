package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	d "github.com/benpate/domain"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// ActorUsername returns a human-readable username for an Actor
func ActorUsername(actor streams.Document) string {

	// If we have a preferred username, then return it as @username@hostname
	if username := actor.PreferredUsername(); username != "" {
		return "@" + username + "@" + d.NameOnly(actor.ID())
	}

	// Otherwise, try to "URL"
	if url := actor.URL(); url != "" {
		return url
	}

	// Otherwise, just punt and use the ID
	return actor.ID()
}

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
func getAuthorization(ctx echo.Context) model.Authorization {

	if sterankoContext, ok := ctx.(*steranko.Context); ok {

		if claims, err := sterankoContext.Authorization(); err == nil {

			if auth, ok := claims.(*model.Authorization); ok {
				return *auth
			}
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

// firstOf is a quickie generic helper that returns the first
// non-zero value from a list of comparable values.
func firstOf[T comparable](values ...T) T {

	var empty T

	// Try each value in the list.  If non-zero, then celebrate success.
	for _, value := range values {
		if value != empty {
			return value
		}
	}

	// Boo, hisss...
	return empty
}

func inlineError(ctx echo.Context, errorMessage string) error {

	header := ctx.Response().Header()
	header.Set("Hx-Reswap", "innerHTML")
	header.Set("Hx-Retarget", "#htmx-response-message")

	return ctx.String(http.StatusOK, `<span class="text-red">`+errorMessage+`</span>`)
}

func closeModalAndRefreshPage(ctx echo.Context) error {
	header := ctx.Response().Header()
	header.Set("Hx-Push-Url", "false")
	header.Set("Hx-Trigger", `{"closeModal": true, "refreshPage": true}`)
	return ctx.NoContent(http.StatusOK)
}

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *domain.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

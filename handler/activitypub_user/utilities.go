package activitypub_user

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return *auth
		}
	}

	return model.NewAuthorization()
}

// getOriginType translates from ActivityStream.Type => model.OriginType constants
func getOriginType(activityType string) string {

	switch activityType {

	case vocab.ActivityTypeAnnounce:
		return model.OriginTypeAnnounce

	case vocab.ActivityTypeLike:
		return model.OriginTypeLike

	case vocab.ActivityTypeDislike:
		return model.OriginTypeDislike
	}

	return model.OriginTypePrimary
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

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *domain.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

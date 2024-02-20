package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
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

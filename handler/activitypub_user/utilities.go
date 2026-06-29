package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/sliceof"
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

// isRecipientPublic returns TRUE if the provided recipients list addresses the
// public audience (in any of the forms that a remote sender might use).
func isRecipientPublic(recipients sliceof.String) bool {
	return recipients.Contains(vocab.NamespaceActivityStreamsPublic) ||
		recipients.Contains(vocab.NamespaceASPublic) ||
		recipients.Contains(vocab.NamespacePublic)
}

// canViewObjectPermissions returns TRUE if the request is allowed to view an Object
// whose recipient list is `permissions`. Public objects are visible to everyone;
// otherwise the requester must prove (via a valid HTTP signature) that they are one
// of the named recipients.
func canViewObjectPermissions(ctx *steranko.Context, factory *service.Factory, permissions sliceof.String) bool {

	// RULE: Public objects are visible to everyone
	if isRecipientPublic(permissions) {
		return true
	}

	// Otherwise, identify the requester via their HTTP signature
	signature, err := sigs.Verify(ctx.Request(), factory.ActivityStream().PublicKeyFinder)

	if err != nil {
		return false
	}

	// RULE: The signed-in actor must be a named recipient of this Object
	return permissions.Contains(signature.ActorID())
}

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *service.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

func getResponseType(ctx *steranko.Context) string {

	switch list.Last(ctx.Request().URL.Path, '/') {

	case "shared":
		return vocab.ActivityTypeAnnounce

	case "liked":
		return vocab.ActivityTypeLike

	case "disliked":
		return vocab.ActivityTypeDislike
	}

	return ""
}

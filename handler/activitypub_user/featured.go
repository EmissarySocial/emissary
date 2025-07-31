package activitypub_user

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

func GetFeaturedCollection(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {
	const location = "handler.activitypub_user.GetFeaturedCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFoundError(location, "User not found")
	}

	// Fallthrough means this is a request for a specific page
	streamService := factory.Stream()
	streams, err := streamService.QueryFeaturedByUser(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading streams")
	}

	// Extract *just* the URL to include in the collection.
	objectIDs := slice.Map(streams, func(stream model.StreamSummary) any {
		return stream.URL
	})

	// Return results to the client.
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.Collection(user.ActivityPubFeaturedURL())
	result.OrderedItems = objectIDs
	result.TotalItems = len(objectIDs)
	result.First = ""

	return ctx.JSON(200, result)
}

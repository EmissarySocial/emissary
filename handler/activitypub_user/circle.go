package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetCircle names one of a User's circles as an audience, and never lists its members
func GetCircle(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetCircle"

	// RULE: A non-public User's circles are not public information
	if !isUserVisible(ctx, user) {
		return derp.NotFound(location, "Circle not found")
	}

	// RULE: A circle is never enumerated, so paging into it is forbidden, not empty
	if isPagingRequest(ctx) {
		return derp.Forbidden(location, "This collection cannot be enumerated")
	}

	// Parse the CircleID from the URL
	circleID, err := primitive.ObjectIDFromHex(ctx.Param("circleId"))

	if err != nil {
		return derp.NotFound(location, "Circle not found")
	}

	// RULE: No circle has the zero ID, and LoadByID would answer it with a validation error
	if circleID.IsZero() {
		return derp.NotFound(location, "Circle not found")
	}

	// RULE: The circle must exist and belong to this User
	circle := model.NewCircle()

	if err := factory.Circle().LoadByID(session, user.UserID, circleID, &circle); err != nil {

		if derp.IsNotFound(err) {
			return derp.NotFound(location, "Circle not found")
		}

		return derp.Wrap(err, location, "Loading circle", "circleId: "+circleID.Hex())
	}

	// Serve the circle under the same URL that addressing names it by
	circleURL := factory.Locator().CircleURL(user.UserID, circle.CircleID)

	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, circleJSONLD(circleURL, user.ActivityPubURL()))
}

// circleJSONLD returns the document that names a circle: its id and its owner, and nothing about its members
func circleJSONLD(circleURL string, ownerURL string) mapof.Any {

	// RULE: No name, no count, and no pages. Every server that receives a circle-only activity
	// learns that the circle exists, and nothing else (OUTBOX2-OMNIBUS §1.11).
	return mapof.Any{
		vocab.AtContext:            vocab.ContextTypeActivityStreams,
		vocab.PropertyID:           circleURL,
		vocab.PropertyType:         vocab.CoreTypeCollection,
		vocab.PropertyAttributedTo: ownerURL,
	}
}

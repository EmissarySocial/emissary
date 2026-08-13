package activitypub_search

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostInbox receives an inbound ActivityPub activity in a SearchQuery's inbox, verifies it, and
// routes it to the matching handler.
func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.PostInbox"

	// Get an ActivityStream service for the Search Domain
	activityService := factory.ActivityStream()
	client := activityService.SearchDomainClient()

	// Create a new request context for the ActivityPub router
	context := Context{
		factory:     factory,
		session:     session,
		stream:      stream,
		searchQuery: searchQuery,
	}

	// Retrieve the activity through the canonical inbox receive funnel (Stage-1 validators + the
	// reserved-namespace sanitizer) -- Stage 1 of the block gate (D5). The owner is NilObjectID
	// (admin-tier rules) -- NOT searchQuery.SearchQueryID, which is a SearchQuery id, not a UserID;
	// passing it would scope the gate to a nonexistent user's rules and silently disable admin
	// blocking here. The verifier keeps signature verification inside Emissary's client stack (BUG-19).
	activity, err := activitypub.ReceiveRequest(ctx.Request(), client, activityService.VerifySignature, factory.Rule(), session, primitive.NilObjectID)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Route the activity to the appropriate handler
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}

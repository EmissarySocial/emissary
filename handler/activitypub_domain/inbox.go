package activitypub_domain

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.PostInbox"

	client := factory.ActivityStream().SearchDomainClient()

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
		session: session,
	}

	// Retrieve the activity through the canonical inbox receive funnel (Stage-1 validators + the
	// reserved-namespace sanitizer), evaluated against admin-tier rules (NilObjectID) -- Stage 1 of
	// the block gate (D5).
	activity, err := activitypub.ReceiveRequest(ctx.Request(), client, factory.Rule(), session, primitive.NilObjectID)

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

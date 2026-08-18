package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service/moderation"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeFlag, vocab.Any, outbox_Flag)
}

// outbox_Flag handles outbound Flag activities (an Emissary user reporting
// external content). It forwards the report to the configured moderation
// backend (e.g. Coop) and also places the activity in the user's outbox for
// delivery to the flagged content's server.
//
// Per the ActivityPub Flag convention, the object is one or more
// bare URLs identifying the reported actor and/or
// their content. The object URL itself serves as the author_id for the
// moderation report — it's the actor or content being flagged.
// See https://www.w3.org/TR/activitystreams-vocabulary/#dfn-flag
// See https://docs.gotosocial.org/en/latest/federation/moderation/
func outbox_Flag(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_Flag"

	// Extract fields from the Flag activity
	actorID := activity.ActorID()      // the reporter (our user)
	objectID := activity.Object().ID() // URL of the reported content/actor
	comment := activity.Content()

	// Forward to the moderation backend
	report := moderation.ReportRequest{
		ActorID:  actorID,
		ObjectID: objectID,
		AuthorID: objectID,
		Comment:  comment,
	}

	if err := context.factory.Moderation().SubmitReport(report); err != nil {
		return derp.Wrap(err, location, "Forwarding Flag to moderation backend", activity.ID())
	}

	// Put the activity into the user's outbox for delivery to the flagged server
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Saving Flag to outbox", activity.ID())
	}

	return context.context.NoContent(http.StatusAccepted)
}

package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/moderation"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeFlag, vocab.Any, inbox_Flag)
}

// inbox_Flag handles inbound Flag activities by forwarding them to the configured
// moderation backend (e.g. Coop). The Flag activity's actor is the reporter, the
// object is the content being reported, and the activity's content is the report
// comment.
// See https://www.w3.org/TR/activitystreams-vocabulary/#dfn-flag
func inbox_Flag(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_Flag"

	// Extract fields from the Flag activity
	actorID := activity.ActorID()
	objectID := activity.Object().ID()
	comment := activity.Content()

	// Try to resolve the flagged object to a local stream so we can include
	// the author and content text in the Coop report. If the object is not
	// local (or resolution fails), AuthorID stays empty.
	var objectContent string
	var authorID string

	streamService := context.factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(context.session, objectID, &stream); err == nil {
		objectContent = stream.Content.HTML
		authorID = stream.AttributedTo.ProfileURL
	}

	// Forward to the moderation backend
	report := moderation.ReportRequest{
		ActorID:       actorID,
		ObjectID:      objectID,
		ObjectContent: objectContent,
		AuthorID:      authorID,
		Comment:       comment,
	}

	if err := context.factory.Moderation().SubmitReport(report); err != nil {
		return derp.Wrap(err, location, "Forwarding Flag to moderation backend", activity.ID())
	}

	return nil
}

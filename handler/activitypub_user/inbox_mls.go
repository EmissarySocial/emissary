package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeGroupInfo, inbox_MLS)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypePrivateMessage, inbox_MLS)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypePublicMessage, inbox_MLS)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeWelcome, inbox_MLS)
}

// inbox_MLS handles all MLS messages received via ActivityPub
func inbox_MLS(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_MLS"

	// The message is stored in the Object
	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.Actor().ID() {
		return derp.Forbidden(location, "MLS message must be attributed to the actor", activity.Value())
	}

	// RULE: The object must be encoded as base64
	if object.Encoding() != vocab.EncodingTypeBase64 {
		return derp.BadRequest(location, "MLS message must be base64-encoded", activity.Value())
	}

	// RULE: The media type must be message/mls
	if object.MediaType() != vocab.MediaTypeMLS {
		return derp.BadRequest(location, "MLS message must have media type message/mls", activity.Value())
	}

	// Populate a new MLSMessage
	mlsMessage := model.NewMLSMessage()
	mlsMessage.UserID = context.user.UserID
	mlsMessage.Content = object.Content()

	// WE'RE NOT STORING OTHER METADATA:
	// i.e. the original object type, actor, attributedTo, or generator

	// Save the MLSMessage to the database
	mlsInboxService := context.factory.MLSInbox()
	if err := mlsInboxService.Save(context.session, &mlsMessage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to store MLS Message")
	}

	return context.context.NoContent(http.StatusOK)
}

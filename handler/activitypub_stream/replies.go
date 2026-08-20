package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/steranko"
)

// GetRepliesCollection serves the Replies collection for a Stream
func GetRepliesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string, stream *model.Stream) error {

	return serveCollection(
		ctx,
		factory,
		session,
		actorID,
		stream,
		model.CollectionTypeReplies,
		stream.ActivityPubRepliesURL(),
	)
}

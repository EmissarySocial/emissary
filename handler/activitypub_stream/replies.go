package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

// GetRepliesCollection serves the Replies collection for a Stream
func GetRepliesCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string, stream *model.Stream) error {

	// RULE: A Stream with closed replies shows none, but keeps them, so reopening restores them (REPLY-CONTROLS §1.2)
	if !stream.AcceptsReplies() {
		return collection.ServeEmpty(ctx, stream.ActivityPubRepliesURL())
	}

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

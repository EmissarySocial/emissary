package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/steranko"
)

func GetContextCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {

	// Compute what a local context URL *should* be
	context := stream.ActivityPubURL() + "/pub/context"

	// RULE: Verify that the stream uses the same context as the one we're serving
	if stream.Context != context {
		return ctx.NoContent(http.StatusNotFound)
	}

	// Serve "the context", which is the full reply chain that matches this context url
	contextService := factory.Context()

	return collection.Serve(ctx,
		context,
		contextService.CollectionCount(session, context, exp.All()),
		contextService.CollectionIterator(session, context, exp.All()),
		collection.WithAttributedTo(stream.AttributedTo.ProfileURL),
	)
}

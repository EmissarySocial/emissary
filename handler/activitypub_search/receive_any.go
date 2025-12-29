package activitypub_search

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	// Wildcard handler to drop any unrecognized activities
	inboxRouter.Add(vocab.Any, vocab.Any, func(context Context, activity streams.Document) error {
		log.Trace().Str("domain", context.factory.Host()).Str("activityType", activity.Type()).Msg("Received unrecognized ActivityPub activity")
		return nil
	})
}

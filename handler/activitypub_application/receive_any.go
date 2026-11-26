package activitypub_application

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

// init registers the wildcard handler that ignores unrecognized activities
func init() {
	inboxRouter.Add(vocab.Any, vocab.Any, func(context Context, activity streams.Document) error {
		log.Trace().Str("domain", context.factory.Host()).Str("activityType", activity.Type()).Msg("Ignoring @application activity")
		return nil
	})
}

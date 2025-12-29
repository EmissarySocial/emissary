package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/rs/zerolog/log"
)

func receive_Unknown(context Context, activity streams.Document) error {
	log.Trace().Str("domain", context.factory.Host()).Str("activityType", activity.Type()).Msg("Received unrecognized ActivityPub activity")
	return nil
}

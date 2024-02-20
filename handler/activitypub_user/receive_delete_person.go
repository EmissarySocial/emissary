package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActorTypePerson, MastodonNOOP)
}

func MastodonNOOP(context Context, document streams.Document) error {
	log.Info().Msg("Ignoring Delete on Remote Person")
	return nil
}

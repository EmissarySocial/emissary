package activitypub

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.ActorTypePerson, MastodonNOOP)
}

func MastodonNOOP(factory *domain.Factory, user *model.User, document streams.Document) error {
	output, _ := json.Marshal(document.Value())
	log.Info().RawJSON("document", output).Msg("Ignoring Delete on Remote Person")
	return nil
}

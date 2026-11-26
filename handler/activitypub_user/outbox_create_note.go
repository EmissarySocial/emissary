package activitypub_user

import (
	"github.com/benpate/hannibal/vocab"
)

// init routes a Create of a Note to the handler that Articles use
func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeNote, outbox_CreateArticle)
}

package activitypub_user

import (
	"github.com/benpate/hannibal/vocab"
)

// outbox_Wildcard handles any ActivityPub activity that doesn't have a specific handler
func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeNote, outbox_CreateArticle)
}

package activitypub_user

import "github.com/benpate/hannibal/vocab"

func init() {
	outboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeNote, outbox_UpdateArticle)
}

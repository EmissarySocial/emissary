package activitypub_user

import "github.com/benpate/hannibal/vocab"

// init registers the handler for outbound Update/Note activities, which share the Article handler
func init() {
	outboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeNote, outbox_UpdateArticle)
}

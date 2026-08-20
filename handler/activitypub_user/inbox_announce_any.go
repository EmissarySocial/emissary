package activitypub_user

import (
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for inbound Announce activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, inbox_LikeOrAnnounce)
}

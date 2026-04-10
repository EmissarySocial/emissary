package activitypub_user

import (
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, inbox_LikeOrAnnounce)
}

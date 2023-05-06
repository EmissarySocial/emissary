package handler

import (
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDislike, vocab.Any, activityPub_LikeOrDislike)
}

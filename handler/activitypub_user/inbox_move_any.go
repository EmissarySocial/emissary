package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/davecgh/go-spew/spew"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeMove, vocab.Any, func(context Context, document streams.Document) error {

		const location = "activitypub_user.Inbox.Move[Any]"

		spew.Dump(location, document.Value())

		//	object := document.Object().LoadLink()
		//	target := document.Object().LoadLink()
		return nil
	})
}

package activitypub_user

import (
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(context Context, document streams.Document) error {

		// Force reload of the cache.  If the document is still there, then it will be refreshed.
		// If the document is gone, then it will be removed from the cache.
		_, _ = context.factory.ActivityStreams().Load(document.Object().ID(), ascache.WithForceReload())

		// Who let the dogs out?
		return nil
	})
}

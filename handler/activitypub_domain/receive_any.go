package activitypub_domain

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/davecgh/go-spew/spew"
)

func init() {
	// Wildcard handler to drop any unrecognized activities
	inboxRouter.Add(vocab.Any, vocab.Any, func(context Context, activity streams.Document) error {
		if canTrace() {
			spew.Dump("RECEIVED UNRECOGNIZED ACTIVITY ---------------------------", activity.Value())
		}
		return nil
	})
}

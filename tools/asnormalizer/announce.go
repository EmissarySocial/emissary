package asnormalizer

import (
	"time"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Announce normalizes an Announce activity
func Announce(document streams.Document) map[string]any {

	return map[string]any{
		"type":      vocab.ActivityTypeAnnounce,
		"id":        document.ID(),
		"actor":     document.ActorID(),
		"object":    document.Object().ID(),
		"published": first(document.Published(), time.Now()),
	}
}

package asnormalizer

import (
	"time"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Dislike normalizes a Dislike activity
func Dislike(document streams.Document) map[string]any {

	return map[string]any{
		"type":      vocab.ActivityTypeDislike,
		"id":        document.ID(),
		"actor":     document.ActorID(),
		"object":    document.Object().ID(),
		"published": first(document.Published(), time.Now()),
	}
}

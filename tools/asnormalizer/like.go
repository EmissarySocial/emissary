package asnormalizer

import (
	"time"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Like normalizes a Like activity into a plain map
func Like(document streams.Document) map[string]any {

	return map[string]any{
		"type":      vocab.ActivityTypeLike,
		"id":        document.ID(),
		"actor":     document.ActorID(),
		"object":    document.Object().ID(),
		"published": first(document.Published(), time.Now()),
	}
}

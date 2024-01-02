package asnormalizer

import (
	"time"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Object normalizes a regular document (Article, Note, etc)
func Object(document streams.Document) map[string]any {

	actual := document.UnwrapActivity()

	// This function is for Articles, Notes, etc.
	// If the actual document is not an object then
	// it must be normalized by someone else.
	if actual.NotObject() {
		return nil
	}

	actorID := first(actual.Actor().ID(), document.Actor().ID())

	return map[string]any{
		vocab.PropertyType:         actual.Type(),
		vocab.PropertyID:           actual.ID(),
		vocab.PropertyActor:        actorID,
		vocab.PropertyAttributedTo: first(actual.AttributedTo().ID(), actorID),
		vocab.PropertyContext:      Context(document),
		vocab.PropertyImage:        Image(actual.Image()),
		vocab.PropertySummary:      actual.Summary(),
		vocab.PropertyContent:      actual.Content(),
		vocab.PropertyPublished:    first(actual.Published(), time.Now()),
	}
}

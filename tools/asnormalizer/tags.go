package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Tags normalizes a slice of Tag values
func Tags(document streams.Document) []map[string]any {

	result := make([]map[string]any, 0, document.Len())

	for tag := range document.Channel() {

		result = append(result, map[string]any{
			vocab.PropertyType: vocab.PropertyTag,
			vocab.PropertyHref: first(tag.Href(), tag.ID()),
			vocab.PropertyName: tag.Name(),
		})
	}

	return result
}

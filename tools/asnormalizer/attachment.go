package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func Attachment(document streams.Document) []map[string]any {

	result := make([]map[string]any, 0, document.Len())

	for attachment := range document.Channel() {

		result = append(result, map[string]any{
			vocab.PropertyType:    attachment.Type(),
			vocab.PropertyName:    attachment.Name(),
			vocab.PropertyContent: first(attachment.Content(), attachment.Get("value").String()),
			vocab.PropertyURL:     first(attachment.URL(), attachment.Href()),
		})
	}

	return result
}

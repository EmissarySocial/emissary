package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func Attachment(document streams.Document) []map[string]any {

	result := make([]map[string]any, 0, document.Len())

	for attachment := range document.Channel() {

		file := map[string]any{
			vocab.PropertyType:      attachment.Type(),
			vocab.PropertyMediaType: attachment.MediaType(),
			vocab.PropertyURL:       first(attachment.URL(), attachment.Href()),
			vocab.PropertyHeight:    attachment.Height(),
			vocab.PropertyWidth:     attachment.Width(),
			vocab.PropertyContent:   first(attachment.Content(), attachment.Get("value").String()),
		}

		result = append(result, file)
	}

	return result
}

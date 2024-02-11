package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Image normalizes an Image object
func Image(image streams.Image) map[string]any {

	return map[string]any{
		vocab.PropertyHref:      first(image.Href(), image.URL()),
		vocab.PropertyHeight:    image.Height(),
		vocab.PropertyWidth:     image.Width(),
		vocab.PropertyMediaType: image.MediaType(),
		vocab.PropertySummary:   image.Summary(),
	}
}

// AttachmentAsImage normalizes an Image object
func AttachmentAsImage(attachment streams.Document) map[string]any {

	return map[string]any{
		vocab.PropertyHref:      attachment.URL(),
		vocab.PropertyHeight:    attachment.Height(),
		vocab.PropertyWidth:     attachment.Width(),
		vocab.PropertyMediaType: attachment.MediaType(),
		vocab.PropertySummary:   attachment.Content(),
	}
}

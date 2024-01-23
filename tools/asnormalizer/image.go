package asnormalizer

import "github.com/benpate/hannibal/streams"

// Image normalizes an Image object
func Image(image streams.Image) map[string]any {

	return map[string]any{
		"href":      first(image.Href(), image.URL()),
		"height":    image.Height(),
		"width":     image.Width(),
		"mediaType": image.MediaType(),
		"summary":   image.Summary(),
	}
}

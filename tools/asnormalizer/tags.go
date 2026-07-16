package asnormalizer

import (
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Tags normalizes a slice of Tag values
func Tags(document streams.Document) []map[string]any {

	result := make([]map[string]any, 0, document.Len())

	for tag := range document.Range() {

		allow := true

		// Do not allow any "internal" tags to be imported from
		// the open web.
		for rel := tag.Rel(); rel.NotNil(); rel = rel.Next() {
			if strings.HasPrefix(rel.String(), "--emissary-") {
				allow = false
				break
			}
		}

		if !allow {
			continue
		}

		// If the tag is allowed, then include it in the result. Preserve the tag's actual TYPE
		// (Hashtag / Mention / Emoji) -- writing the literal "tag" property name here erased it, which
		// made Hashtags indistinguishable from Mentions downstream (D12; the TAG rule engine needs it).
		result = append(result, map[string]any{
			vocab.PropertyType: tag.Type(),
			vocab.PropertyHref: first(tag.Href(), tag.ID()),
			vocab.PropertyName: tag.Name(),
		})
	}

	return result
}

package asnormalizer

import (
	"net/url"
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// emojiTagType is the ActivityStreams type of a custom Emoji tag.
const emojiTagType = "Emoji"

// isWebURL returns TRUE if the value is an http or https URL with a host.
func isWebURL(value string) bool {

	parsed, err := url.Parse(value)

	if err != nil {
		return false
	}

	return (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

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
		normalized := map[string]any{
			vocab.PropertyType: tag.Type(),
			vocab.PropertyHref: first(tag.Href(), tag.ID()),
			vocab.PropertyName: tag.Name(),
		}

		// A custom Emoji is only useful with its image, so keep that one web address
		if tag.Type() == emojiTagType {
			if icon := tag.Icon().URL(); isWebURL(icon) {
				normalized[vocab.PropertyIcon] = icon
			}
		}

		result = append(result, normalized)
	}

	return result
}

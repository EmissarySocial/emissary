package mastodon

import (
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/toot/object"
)

// labelFilterContexts are the places a labelled post is shown behind its warning
var labelFilterContexts = []string{"home", "notifications", "public", "thread", "account"}

// labelFilterResults turns the viewer's annotating rule labels on a document into "warn" filter results
func labelFilterResults(labels metadata.LabelSet) []object.FilterResult {

	// Apps show the post behind a "filtered" warning. A hiding label (a block or a mute) keeps the
	// post out of the timeline altogether, so it is not listed here.

	annotations := labels.Annotations()

	if len(annotations) == 0 {
		return nil
	}

	result := make([]object.FilterResult, 0, len(annotations))

	for _, label := range annotations {

		if label.Value == "" {
			continue
		}

		// RULE: The ID only needs to be stable for the same label, so apps can tell results apart
		id := label.Href

		if id == "" {
			id = "label:" + label.Value
		}

		result = append(result, object.FilterResult{
			Filter: object.Filter{
				ID:           id,
				Title:        label.Value,
				Context:      labelFilterContexts,
				FilterAction: "warn",
			},
		})
	}

	return result
}

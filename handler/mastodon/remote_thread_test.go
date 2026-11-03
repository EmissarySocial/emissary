package mastodon

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// stepper builds a walkThread step from a table of post -> parent, recording each address it is asked for.
func stepper(parents map[string]string, asked *[]string) func(string) (object.Status, string, bool) {

	return func(url string) (object.Status, string, bool) {

		*asked = append(*asked, url)
		parent, found := parents[url]

		if !found {
			return object.Status{}, "", false
		}

		return object.Status{URI: url}, parent, true
	}
}

// uris returns the address of each Status, in order.
func uris(statuses []object.Status) []string {

	result := []string{}

	for _, status := range statuses {
		result = append(result, status.URI)
	}

	return result
}

// TestWalkThread_ListsAncestorsOldestFirst confirms a chain comes back with the root first.
func TestWalkThread_ListsAncestorsOldestFirst(t *testing.T) {

	asked := []string{}
	parents := map[string]string{"c": "b", "b": "a", "a": ""}

	require.Equal(t, []string{"a", "b", "c"}, uris(walkThread("c", 20, stepper(parents, &asked))))
	require.Equal(t, []string{"c", "b", "a"}, asked, "it climbs one post at a time")
}

// TestWalkThread_StopsWhereItShould covers a depth limit, an unreadable post, a loop, and no starting post.
func TestWalkThread_StopsWhereItShould(t *testing.T) {

	asked := []string{}

	long := map[string]string{"e": "d", "d": "c", "c": "b", "b": "a", "a": ""}
	require.Equal(t, []string{"d", "e"}, uris(walkThread("e", 2, stepper(long, &asked))), "only the nearest posts, still oldest first")

	asked = []string{}
	gap := map[string]string{"c": "b", "b": "missing"}
	require.Equal(t, []string{"b", "c"}, uris(walkThread("c", 20, stepper(gap, &asked))), "an unreadable parent ends the thread without losing what was read")

	asked = []string{}
	loop := map[string]string{"a": "b", "b": "a"}
	require.Equal(t, []string{"b", "a"}, uris(walkThread("a", 20, stepper(loop, &asked))), "a post that replies to itself is visited once")
	require.Len(t, asked, 2)

	asked = []string{}
	require.Empty(t, walkThread("", 20, stepper(nil, &asked)))
	require.Empty(t, asked, "no starting address means nothing is fetched")

	require.Empty(t, walkThread("x", 20, stepper(map[string]string{}, &asked)), "an unreadable first post gives an empty thread, not an error")
}

// TestDocumentAuthorURL covers attributedTo, the actor fallback, and neither.
func TestDocumentAuthorURL(t *testing.T) {

	byAttribution := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/1", "attributedTo": "https://example.com/users/ben", "actor": "https://example.com/users/other"})
	require.Equal(t, "https://example.com/users/ben", documentAuthorURL(byAttribution))

	byActor := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/2", "actor": "https://example.com/users/amy"})
	require.Equal(t, "https://example.com/users/amy", documentAuthorURL(byActor))

	neither := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/3"})
	require.Equal(t, "", documentAuthorURL(neither))
}

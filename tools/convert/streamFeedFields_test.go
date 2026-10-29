package convert

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamFeedFields_ProjectionKeepsFeedOutput confirms that a Stream loaded with only
// StreamFeedFields converts to the same feed items as the whole record
func TestStreamFeedFields_ProjectionKeepsFeedOutput(t *testing.T) {

	stream := model.NewStream()
	stream.URL = "https://example.com/blog/post"
	stream.Token = "post"
	stream.Label = "A Post"
	stream.Summary = "The summary"
	stream.IconURL = "https://example.com/icon.png"
	stream.AttributedTo = model.PersonLink{UserID: primitive.NewObjectID(), Name: "Author", ProfileURL: "https://example.com/@author", IconURL: "https://example.com/author.png"}
	stream.Content = model.Content{Format: "HTML", Raw: "<p>Body</p>", HTML: "<p>Body</p>"}
	stream.PublishDate = 1_700_000_000
	stream.UpdateDate = 1_700_000_500_000
	stream.Hashtags = []string{"unrelated"}
	stream.Data = map[string]any{"unrelated": "value", "enclosureUrl": "https://example.com/blog/post/audio", "enclosureType": "audio/ogg"}

	projected := projectStream(t, stream, StreamFeedFields())

	require.True(t, sameFeedItems(stream, projected), "a projected Stream must convert exactly as the whole record")

	// The projection must really drop fields, or this test proves nothing
	require.Empty(t, projected.Hashtags)
	require.Empty(t, projected.Content.Raw)

	// Every listed field is read by a converter, so leaving any one out changes a feed item
	for _, field := range StreamFeedFields() {

		without := slices.DeleteFunc(StreamFeedFields(), func(name string) bool { return name == field })
		partial := projectStream(t, stream, without)

		require.False(t, sameFeedItems(stream, partial), "field %q is listed but no converter reads it", field)
	}
}

// sameFeedItems returns TRUE if both Streams convert to equal RSS/Atom and JSON Feed items
func sameFeedItems(first model.Stream, second model.Stream) bool {
	return reflect.DeepEqual(StreamToGorillaFeed(first), StreamToGorillaFeed(second)) &&
		reflect.DeepEqual(StreamToJsonFeed(first), StreamToJsonFeed(second))
}

// projectStream round-trips a Stream through BSON, keeping only the named fields (top-level, or one level
// deep as "parent.child"), as a MongoDB projection would
func projectStream(t *testing.T, stream model.Stream, fields []string) model.Stream {

	encoded, err := bson.Marshal(stream)
	require.NoError(t, err)

	document := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &document))

	projected := bson.M{}

	for _, field := range fields {

		parent, child, isNested := strings.Cut(field, ".")

		if !isNested {
			if value, exists := document[field]; exists {
				projected[field] = value
			}
			continue
		}

		source, isDocument := document[parent].(bson.M)
		require.True(t, isDocument, "field %q is not a subdocument", parent)

		target, _ := projected[parent].(bson.M)
		if target == nil {
			target = bson.M{}
			projected[parent] = target
		}

		target[child] = source[child]
	}

	encoded, err = bson.Marshal(projected)
	require.NoError(t, err)

	result := model.NewStream()
	require.NoError(t, bson.Unmarshal(encoded, &result))

	return result
}

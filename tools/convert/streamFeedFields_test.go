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
	stream.Token.Set("post")
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

	// A page that never entered the outbox has no publish date, and is dated by its create date
	undated := stream
	undated.PublishDate = 0
	undated.CreateDate = 1_600_000_000_000

	require.True(t, sameFeedItems(undated, projectStream(t, undated, StreamFeedFields())))

	// Every listed field is read by a converter, so leaving any one out changes a feed item,
	// for a Stream with a publish date or for one without
	for _, field := range StreamFeedFields() {

		without := slices.DeleteFunc(StreamFeedFields(), func(name string) bool { return name == field })

		isRead := !sameFeedItems(stream, projectStream(t, stream, without)) ||
			!sameFeedItems(undated, projectStream(t, undated, without))

		require.True(t, isRead, "field %q is listed but no converter reads it", field)
	}
}

// TestStreamFeed_DatesAnUnsentPage confirms that a feed item shows the create date of a page
// that has no publish date, and never the start of 1970
func TestStreamFeed_DatesAnUnsentPage(t *testing.T) {

	stream := model.NewStream()
	stream.CreateDate = 1_600_000_000_000

	require.Equal(t, int64(1_600_000_000), StreamToGorillaFeed(stream).Created.Unix())
	require.Equal(t, int64(1_600_000_000), StreamToJsonFeed(stream).DatePublished.Unix())

	// A publish date wins when there is one
	stream.PublishDate = 1_700_000_000

	require.Equal(t, int64(1_700_000_000), StreamToGorillaFeed(stream).Created.Unix())
	require.Equal(t, int64(1_700_000_000), StreamToJsonFeed(stream).DatePublished.Unix())
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

package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestFeedOrder confirms that a feed lists its Streams newest first by the date each one
// shows, so a page with no publish date takes its place by its create date
func TestFeedOrder(t *testing.T) {

	newStream := func(label string, publishDate int64, createDate int64) model.Stream {
		stream := model.NewStream()
		stream.Label = label
		stream.PublishDate = publishDate
		stream.CreateDate = createDate
		return stream
	}

	streams := feedOrder([]model.Stream{
		newStream("old note", 1_000, 1_000_000),
		newStream("new article, never sent", 0, 3_000_000),
		newStream("backdated", 500, 4_000_000),
		newStream("newer note", 2_000, 2_000_000),
	})

	labels := make([]string, 0, len(streams))

	for _, stream := range streams {
		labels = append(labels, stream.Label)
	}

	require.Equal(t, []string{"new article, never sent", "newer note", "old note", "backdated"}, labels)

	// An empty feed stays empty
	require.Empty(t, feedOrder(nil))
}

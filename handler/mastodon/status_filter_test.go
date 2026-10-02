package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// TestStatusFilter confirms the exclude_replies and only_media filters
func TestStatusFilter(t *testing.T) {

	plain := object.Status{ID: "plain"}
	withMedia := object.Status{ID: "media", MediaAttachments: []object.MediaAttachment{{ID: "m"}}}
	replyToOther := object.Status{ID: "reply-other", InReplyToID: "p", InReplyToAccountID: "someone-else"}
	replyToSelf := object.Status{ID: "reply-self", InReplyToID: "p", InReplyToAccountID: "me"}
	replyUnknown := object.Status{ID: "reply-unknown", InReplyToID: "p"}

	noFilter := statusFilter(false, false, "me")
	require.True(t, noFilter(plain))
	require.True(t, noFilter(replyToOther))

	noReplies := statusFilter(true, false, "me")
	require.True(t, noReplies(plain))
	require.False(t, noReplies(replyToOther), "a reply to someone else is excluded")
	require.True(t, noReplies(replyToSelf), "a reply to your own post stays in")
	require.False(t, noReplies(replyUnknown), "a reply whose author is unknown counts as a reply to someone else")

	mediaOnly := statusFilter(false, true, "me")
	require.True(t, mediaOnly(withMedia))
	require.False(t, mediaOnly(plain))
}

// TestCollectFiltered confirms a page is filled from several batches, and stops when posts run out
func TestCollectFiltered(t *testing.T) {

	// 10 posts, newest first (createDate 10 down to 1); only the even ones pass
	all := []model.Stream{}

	for date := int64(10); date >= 1; date-- {
		post := model.NewStream()
		post.CreateDate = date
		all = append(all, post)
	}

	fetchCalls := 0
	batchSize := 3 // a real fetch asks for exactly the page limit

	fetch := func(before int64) ([]model.Stream, error) {

		fetchCalls++
		batch := []model.Stream{}

		for _, post := range all {
			if (before == 0 || post.CreateDate < before) && len(batch) < batchSize {
				batch = append(batch, post)
			}
		}

		return batch, nil
	}

	convert := func(streams []model.Stream) []object.Status {

		statuses := make([]object.Status, len(streams))

		for index, stream := range streams {
			statuses[index] = object.Status{ID: string(rune('a' + stream.CreateDate))}
			if stream.CreateDate%2 == 0 {
				statuses[index].MediaAttachments = []object.MediaAttachment{{ID: "m"}}
			}
		}

		return statuses
	}

	keep := statusFilter(false, true, "me")

	streams, statuses, err := collectFiltered(3, fetch, convert, keep)
	require.NoError(t, err)
	require.Len(t, statuses, 3, "keeps reading batches until the page is full")
	require.Equal(t, []int64{10, 8, 6}, []int64{streams[0].CreateDate, streams[1].CreateDate, streams[2].CreateDate}, "newest first, in step with the statuses")

	fetchCalls = 0
	batchSize = 10
	streams, _, err = collectFiltered(10, fetch, convert, keep)
	require.NoError(t, err)
	require.Len(t, streams, 5, "returns every post that passes when fewer than a page exist")
	require.Equal(t, 2, fetchCalls, "reads until a batch comes back empty")
}

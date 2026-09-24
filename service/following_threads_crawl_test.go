package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// storedNewsItem returns a NewsItem already saved from the given Following URL, in the given state
func storedNewsItem(followingURL string, stateID string) model.NewsItem {

	result := model.NewNewsItem()
	result.URL = "https://example.com/post/1"
	result.StateID = stateID
	result.AddReference(model.OriginLink{URL: followingURL, Type: model.OriginTypePrimary})

	return result
}

// arrivingNewsItem returns a copy of the stored post arriving from the given Following URL
func arrivingNewsItem(followingURL string, originType string) model.NewsItem {

	result := model.NewNewsItem()
	result.URL = "https://example.com/post/1"
	result.AddReference(model.OriginLink{URL: followingURL, Type: originType})

	return result
}

/******************************************
 * mergeNewsItem
 ******************************************/

// A poll re-reading an unchanged post from the same Following changes nothing.
func TestMergeNewsItem_SameOriginIsUnchanged(t *testing.T) {

	previous := storedNewsItem("https://alice.example/", model.NewsItemStateRead)
	save := mergeNewsItem(&previous, arrivingNewsItem("https://alice.example/", model.OriginTypePrimary))

	require.Equal(t, newsItemUnchanged, save)
}

// The same post arriving through a different Following adds an origin.
func TestMergeNewsItem_NewOriginIsAdded(t *testing.T) {

	previous := storedNewsItem("https://alice.example/", model.NewsItemStateRead)
	save := mergeNewsItem(&previous, arrivingNewsItem("https://bob.example/", model.OriginTypePrimary))

	require.Equal(t, newsItemOriginAdded, save)
}

// A reply on a READ post flags it as NEW-REPLIES.
func TestMergeNewsItem_ReplyMarksReadItem(t *testing.T) {

	previous := storedNewsItem("https://alice.example/", model.NewsItemStateRead)
	save := mergeNewsItem(&previous, arrivingNewsItem("https://alice.example/", model.OriginTypeReply))

	require.Equal(t, newsItemMarkedNewReplies, save)
	require.Equal(t, model.NewsItemStateNewReplies, previous.StateID)
}

// A reply on a post that is already flagged, unread, or muted changes nothing stored.
func TestMergeNewsItem_ReplyLeavesOtherStates(t *testing.T) {

	for _, stateID := range []string{model.NewsItemStateNewReplies, model.NewsItemStateUnread, model.NewsItemStateMuted} {

		previous := storedNewsItem("https://alice.example/", stateID)
		save := mergeNewsItem(&previous, arrivingNewsItem("https://alice.example/", model.OriginTypeReply))

		require.Equal(t, newsItemUnchanged, save, stateID)
		require.Equal(t, stateID, previous.StateID, stateID)
	}
}

/******************************************
 * shouldCrawl (one test per rule, BUG-183 §12)
 ******************************************/

// Rule 1: a muted thread is never crawled, even when it gains an origin.
func TestShouldCrawl_MutedIsNeverCrawled(t *testing.T) {

	require.False(t, shouldCrawl(newsItemOriginAdded, model.NewsItemStateMuted, true, true))
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateMuted, true, true))
}

// Rule 2: a post never seen before is crawled.
func TestShouldCrawl_CreatedIsCrawled(t *testing.T) {
	require.True(t, shouldCrawl(newsItemCreated, model.NewsItemStateUnread, false, true))
}

// Rule 3: a post arriving through another Following is crawled.
func TestShouldCrawl_OriginAddedIsCrawled(t *testing.T) {
	require.True(t, shouldCrawl(newsItemOriginAdded, model.NewsItemStateRead, false, false))
}

// Rule 4: a new reply on a read thread is crawled.
func TestShouldCrawl_MarkedNewRepliesIsCrawled(t *testing.T) {
	require.True(t, shouldCrawl(newsItemMarkedNewReplies, model.NewsItemStateNewReplies, true, true))
}

// Rule 5: a fresh reply on a thread already flagged, or not yet read, is crawled.
func TestShouldCrawl_FreshReplyOnFlaggedThreadIsCrawled(t *testing.T) {
	require.True(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateNewReplies, true, true))
	require.True(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateUnread, true, true))
}

// Rule 6: a poll re-reading a reply from the cache is not crawled, whatever its state.
func TestShouldCrawl_CachedReplyIsNotCrawled(t *testing.T) {
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateNewReplies, true, false))
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateUnread, true, false))
}

// Rule 6: an unchanged post that is not a reply is not crawled, even when fresh.
func TestShouldCrawl_UnchangedPrimaryIsNotCrawled(t *testing.T) {
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateUnread, false, true))
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateRead, false, false))
}

// Rule 6: a fresh reply on a READ thread that did not change it is not crawled.  (Unreachable
// today, because a reply always flags a READ thread; pinned so rule 5 cannot quietly widen.)
func TestShouldCrawl_UnchangedReadIsNotCrawled(t *testing.T) {
	require.False(t, shouldCrawl(newsItemUnchanged, model.NewsItemStateRead, true, true))
}

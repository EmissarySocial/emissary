package consumer

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestCrawlDownReplyTree_DepthCap pins the recursion bound that keeps the reply-tree
// crawl finite: a task at (or past) maxCrawlDepth exits successfully BEFORE touching
// the factory (nil here proves it), so a cycle in remote reply data can never breed
// tasks beyond the cap.
func TestCrawlDownReplyTree_DepthCap(t *testing.T) {

	// A task exactly at the cap stops without any I/O.
	result := CrawlDownReplyTree(nil, mapof.Any{
		"url":   "https://example.com/note/1",
		"depth": maxCrawlDepth,
	})

	require.Equal(t, queue.ResultStatusSuccess, result.Status)
	require.NoError(t, result.Error)

	// A task past the cap (defensive: should never be produced) also stops.
	result = CrawlDownReplyTree(nil, mapof.Any{
		"url":   "https://example.com/note/1",
		"depth": maxCrawlDepth + 1,
	})

	require.Equal(t, queue.ResultStatusSuccess, result.Status)
	require.NoError(t, result.Error)
}

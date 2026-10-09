package mastodon

import (
	"testing"

	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// node returns a thread entry for a post address and publish time.
func node(url string, published int64) threadNode {
	return threadNode{URL: url, Published: published, Build: func() object.Status { return object.Status{URI: url} }}
}

// nodeURLs returns the address of each thread entry, in order.
func nodeURLs(nodes []threadNode) []string {

	result := []string{}

	for _, item := range nodes {
		result = append(result, item.URL)
	}

	return result
}

// TestFlattenThread_OrdersEachBranchByTime confirms replies come oldest first, each followed by its own replies.
func TestFlattenThread_OrdersEachBranchByTime(t *testing.T) {

	tree := map[string][]threadNode{
		"root":  {node("late", 30), node("early", 10)},
		"early": {node("early-b", 22), node("early-a", 21)},
		"late":  {node("late-a", 31)},
	}

	childrenOf := func(url string) []threadNode { return append([]threadNode{}, tree[url]...) }

	require.Equal(t, []string{"early", "early-a", "early-b", "late", "late-a"}, nodeURLs(flattenThread("root", childrenOf, 60, 20)))
}

// TestFlattenThread_StopsAtLimitsAndLoops covers the post limit, the depth limit, a post listed twice, and a loop back to the root.
func TestFlattenThread_StopsAtLimitsAndLoops(t *testing.T) {

	tree := map[string][]threadNode{
		"root": {node("a", 1), node("b", 2), node("c", 3)},
		"a":    {node("a1", 4)},
		"a1":   {node("a2", 5)},
	}

	childrenOf := func(url string) []threadNode { return append([]threadNode{}, tree[url]...) }

	require.Equal(t, []string{"a", "a1"}, nodeURLs(flattenThread("root", childrenOf, 2, 20)), "limit counts every post")
	require.Equal(t, []string{"a", "a1", "b", "c"}, nodeURLs(flattenThread("root", childrenOf, 60, 2)), "depth 2 reaches replies to replies, no deeper")

	repeats := map[string][]threadNode{
		"root": {node("a", 1), node("b", 2)},
		"a":    {node("shared", 3), node("root", 4)},
		"b":    {node("shared", 5)},
	}

	repeatsOf := func(url string) []threadNode { return append([]threadNode{}, repeats[url]...) }

	require.Equal(t, []string{"a", "shared", "b"}, nodeURLs(flattenThread("root", repeatsOf, 60, 20)), "a post is shown once, and the root never reappears")
	require.Empty(t, flattenThread("nobody", repeatsOf, 60, 20))
}

// TestConversationCandidates_ReadsTheContextCollection confirms the posts a server lists for a conversation are all returned.
func TestConversationCandidates_ReadsTheContextCollection(t *testing.T) {

	client := &fakeClient{documents: map[string]map[string]any{
		"https://a.example.com/contexts/1": {
			"id": "https://a.example.com/contexts/1", "type": "Collection",
			"first": map[string]any{"type": "CollectionPage", "items": []any{"https://a.example.com/p/1", "https://a.example.com/p/2"}, "next": "https://a.example.com/contexts/1?page=2"},
		},
		"https://a.example.com/contexts/1?page=2": {
			"id": "https://a.example.com/contexts/1?page=2", "type": "CollectionPage", "items": []any{"https://b.example.com/p/3"},
		},
		"https://a.example.com/p/1": {"id": "https://a.example.com/p/1", "type": "Note", "context": "https://a.example.com/contexts/1"},
	}}

	post, err := client.Load("https://a.example.com/p/1")
	require.NoError(t, err)

	require.Equal(t, []string{"https://a.example.com/p/1", "https://a.example.com/p/2", "https://b.example.com/p/3"}, conversationCandidates(client, post))
}

// TestThreadCandidates_FallsBackToTheRepliesCollection covers a conversation that cannot be read, using
// Mastodon's replies shape: an empty inline first page that points to the page holding the replies.
func TestThreadCandidates_FallsBackToTheRepliesCollection(t *testing.T) {

	client := &fakeClient{documents: map[string]map[string]any{
		"https://a.example.com/p/1": {
			"id": "https://a.example.com/p/1", "type": "Note",
			"context": "tag:a.example,2026-10-01:objectId=1:objectType=Conversation",
			"replies": "https://a.example.com/p/1/replies",
		},
		"https://a.example.com/p/1/replies": {
			"id": "https://a.example.com/p/1/replies", "type": "Collection",
			"first": map[string]any{"type": "CollectionPage", "next": "https://a.example.com/p/1/replies?page=true", "items": []any{}},
		},
		"https://a.example.com/p/1/replies?page=true": {
			"id": "https://a.example.com/p/1/replies?page=true", "type": "CollectionPage",
			"items": []any{map[string]any{"id": "https://b.example.com/p/2", "type": "Note", "inReplyTo": "https://a.example.com/p/1"}},
		},
		"https://b.example.com/p/2": {"id": "https://b.example.com/p/2", "type": "Note", "inReplyTo": "https://a.example.com/p/1", "replies": "https://b.example.com/p/2/replies"},
		"https://b.example.com/p/2/replies": {
			"id": "https://b.example.com/p/2/replies", "type": "Collection",
			"first": map[string]any{"type": "CollectionPage", "items": []any{"https://c.example.com/p/3"}},
		},
		"https://c.example.com/p/3": {"id": "https://c.example.com/p/3", "type": "Note", "inReplyTo": "https://b.example.com/p/2"},
	}}

	post, err := client.Load("https://a.example.com/p/1")
	require.NoError(t, err)

	require.Equal(t, []string{"https://b.example.com/p/2", "https://c.example.com/p/3"}, threadCandidates(client, post), "replies, then the reply's own replies")
}

// TestThreadCandidates_PrefersTheConversationWhenItListsPosts confirms the context collection wins over walking replies.
func TestThreadCandidates_PrefersTheConversationWhenItListsPosts(t *testing.T) {

	client := &fakeClient{documents: map[string]map[string]any{
		"https://a.example.com/p/1": {"id": "https://a.example.com/p/1", "type": "Note", "context": "https://a.example.com/contexts/1", "replies": "https://a.example.com/p/1/replies"},
		"https://a.example.com/contexts/1": {
			"id": "https://a.example.com/contexts/1", "type": "Collection",
			"first": map[string]any{"type": "CollectionPage", "items": []any{"https://a.example.com/p/1", "https://a.example.com/p/2"}},
		},
	}}

	post, err := client.Load("https://a.example.com/p/1")
	require.NoError(t, err)

	require.Equal(t, []string{"https://a.example.com/p/1", "https://a.example.com/p/2"}, threadCandidates(client, post))
}

// TestLoadThreadDocuments_LeavesOutWhatCannotBeRead confirms an unreadable post is dropped and the rest are kept.
func TestLoadThreadDocuments_LeavesOutWhatCannotBeRead(t *testing.T) {

	client := &fakeClient{documents: map[string]map[string]any{
		"https://a.example.com/p/1": {"id": "https://a.example.com/p/1", "type": "Note"},
		"https://a.example.com/p/3": {"id": "https://a.example.com/p/3", "type": "Note"},
	}}

	documents := loadThreadDocuments(client, []string{"https://a.example.com/p/1", "https://a.example.com/p/2", "https://a.example.com/p/3"})

	require.Len(t, documents, 2)
	require.Contains(t, documents, "https://a.example.com/p/1")
	require.Contains(t, documents, "https://a.example.com/p/3")
	require.NotContains(t, documents, "https://a.example.com/p/2")
}

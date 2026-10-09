package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// testPost returns a Stream with the given address that replies to inReplyTo.
func testPost(url string, inReplyTo string) model.Stream {
	stream := model.NewStream()
	stream.URL = url
	stream.InReplyTo = inReplyTo
	return stream
}

// urlsOf returns the address of each Stream, in order.
func urlsOf(streams []model.Stream) []string {

	result := []string{}

	for _, stream := range streams {
		result = append(result, stream.URL)
	}

	return result
}

// TestFlattenReplies confirms replies come back in thread order, with limits and loops respected
func TestFlattenReplies(t *testing.T) {

	children := map[string][]model.Stream{
		"root": {testPost("a", "root"), testPost("b", "root")},
		"a":    {testPost("c", "a")},
		"c":    {testPost("d", "c")},
	}

	repliesOf := func(url string) []model.Stream { return children[url] }

	require.Equal(t, []string{"a", "c", "d", "b"}, urlsOf(flattenReplies("root", repliesOf, 10, 10)), "each reply is followed by its own replies")
	require.Equal(t, []string{"a", "c"}, urlsOf(flattenReplies("root", repliesOf, 2, 10)), "stops at the limit")
	require.Equal(t, []string{"a", "b"}, urlsOf(flattenReplies("root", repliesOf, 10, 1)), "stops at the maximum depth")
	require.Empty(t, flattenReplies("nobody", repliesOf, 10, 10))

	children["d"] = []model.Stream{testPost("a", "d")}
	require.Equal(t, []string{"a", "c", "d", "b"}, urlsOf(flattenReplies("root", repliesOf, 10, 10)), "a loop back to a seen post is ignored")
}

// TestWalkAncestors confirms parents come back root first, stopping at the first one that is missing
func TestWalkAncestors(t *testing.T) {

	posts := map[string]model.Stream{
		"grandparent": testPost("grandparent", "elsewhere"),
		"parent":      testPost("parent", "grandparent"),
	}

	parentOf := func(url string) *model.Stream {
		if post, found := posts[url]; found {
			return &post
		}
		return nil
	}

	start := testPost("child", "parent")

	require.Equal(t, []string{"grandparent", "parent"}, urlsOf(walkAncestors(&start, parentOf, 10)), "root first, stopping at a parent that is not on this server")
	require.Equal(t, []string{"parent"}, urlsOf(walkAncestors(&start, parentOf, 1)), "stops at the limit")

	orphan := testPost("orphan", "")
	require.Empty(t, walkAncestors(&orphan, parentOf, 10))

	posts["grandparent"] = testPost("grandparent", "child")
	require.Equal(t, []string{"grandparent", "parent"}, urlsOf(walkAncestors(&start, parentOf, 10)), "a loop back to the start is ignored")
}

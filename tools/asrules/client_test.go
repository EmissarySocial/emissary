package asrules

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// fakeInner is a streams.Client that records the calls it receives, standing in for the network stack.
type fakeInner struct {
	loaded  []string
	saved   int
	deleted int
}

// SetRootClient implements the streams.Client interface. The stub ignores the root client.
func (c *fakeInner) SetRootClient(streams.Client) {}

// Load implements the streams.Client interface, recording the URI and returning a bare document
func (c *fakeInner) Load(uri string, _ ...any) (streams.Document, error) {
	c.loaded = append(c.loaded, uri)
	return streams.NewDocument(map[string]any{vocab.PropertyID: uri}), nil
}

// Save implements the streams.Client interface, counting the call
func (c *fakeInner) Save(streams.Document) error { c.saved++; return nil }

// Delete implements the streams.Client interface, counting the call
func (c *fakeInner) Delete(string) error { c.deleted++; return nil }

// checkerFunc builds a Checker returning `pre` for the URL-only call (NilDocument) and `post` once
// a document has loaded.
func checkerFunc(pre metadata.LabelSet, post metadata.LabelSet) Checker {
	return func(_ string, document streams.Document) (metadata.LabelSet, error) {
		if document.IsNil() {
			return pre, nil
		}
		return post, nil
	}
}

// hiddenLabels builds a LabelSet containing a single hidden label with the provided reason
func hiddenLabels(reason string) metadata.LabelSet {
	return metadata.LabelSet{{Value: reason, IsHidden: true}}
}

// annotationLabels builds a LabelSet containing one visible label per provided value
func annotationLabels(values ...string) metadata.LabelSet {
	result := make(metadata.LabelSet, 0, len(values))
	for _, value := range values {
		result = append(result, metadata.Label{Value: value})
	}
	return result
}

// A hidden (blocked) URL is refused before the network stack, and the refusal still carries the
// verdict in its Metadata for the placeholder.
func TestClient_HiddenRefused(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(hiddenLabels("Blocked by server policy"), nil))

	document, err := client.Load("https://evil.example/post")

	require.NotNil(t, err)
	require.True(t, document.IsNil())
	require.Empty(t, inner.loaded)
	require.True(t, document.Metadata.IsRuleHidden())
	require.Equal(t, "Blocked by server policy", document.Metadata.Labels.Reason())
}

// A muted URL is refused just like a blocked one -- inbound, mute and block hide identically.
func TestClient_MuteAlsoRefused(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(hiddenLabels("Muted by your rules"), nil))

	document, err := client.Load("https://gray.example/post")

	require.NotNil(t, err)
	require.True(t, document.IsNil())
	require.Empty(t, inner.loaded)
	require.Equal(t, "Muted by your rules", document.Metadata.Labels.Reason())
}

// WithReveal lets a hidden document through the gate, still stamped with its verdict.
func TestClient_HiddenRevealed(t *testing.T) {

	inner := &fakeInner{}
	blocked := hiddenLabels("Blocked by your rules")
	client := New(inner, checkerFunc(blocked, blocked))

	document, err := client.Load("https://evil.example/post", WithReveal(true))

	require.Nil(t, err)
	require.Equal(t, []string{"https://evil.example/post"}, inner.loaded)
	require.True(t, document.Metadata.IsRuleHidden())
	require.Equal(t, "Blocked by your rules", document.Metadata.Labels.Reason())
}

// A clean document passes through with clean Metadata.
func TestClient_CleanPassthrough(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(nil, nil))

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.Equal(t, "https://good.example/post", document.ID())
	require.Equal(t, []string{"https://good.example/post"}, inner.loaded)
	require.False(t, document.Metadata.IsRuleHidden())
	require.False(t, document.Metadata.Labels.HasAnnotations())
}

// The post-load evaluation adds annotations that the URL alone could not see (author, tags).
func TestClient_PostLoadAddsAnnotations(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(nil, annotationLabels("Spam", "Politics")))

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.False(t, document.Metadata.IsRuleHidden())
	require.True(t, document.Metadata.Labels.HasAnnotations())
	require.Len(t, document.Metadata.Labels.Annotations(), 2)
}

// A URL clean at fetch time but hidden once loaded (an author-level mute) is fetched, then stamped
// hidden -- this is how a muted author's replies collapse in a thread.
func TestClient_PostLoadHidesByAuthor(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(nil, hiddenLabels("Muted by your rules")))

	document, err := client.Load("https://good.example/reply")

	require.Nil(t, err)
	require.Equal(t, []string{"https://good.example/reply"}, inner.loaded)
	require.True(t, document.Metadata.IsRuleHidden())
}

// A pre-fetch checker error fails OPEN: the fetch proceeds, and the post-load verdict stamps.
func TestClient_PreCheckerErrorFailsOpen(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(_ string, document streams.Document) (metadata.LabelSet, error) {
		if document.IsNil() {
			return nil, derp.Internal("test", "database down")
		}
		return annotationLabels("Late"), nil
	})

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.Equal(t, []string{"https://good.example/post"}, inner.loaded)
	require.Len(t, document.Metadata.Labels.Annotations(), 1)
}

// A post-load checker error also fails open, keeping the URL-level verdict already in hand.
func TestClient_PostCheckerErrorKeepsURLVerdict(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(_ string, document streams.Document) (metadata.LabelSet, error) {
		if document.IsNil() {
			return annotationLabels("Politics"), nil
		}
		return nil, derp.Internal("test", "database down")
	})

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.Equal(t, "Politics", document.Metadata.Labels.Annotations()[0].Value)
}

// Save and Delete pass through unchanged -- rules gate reads, not cache writes.
func TestClient_SaveDeletePassthrough(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, checkerFunc(hiddenLabels("Blocked"), nil))

	require.Nil(t, client.Save(streams.NilDocument()))
	require.Nil(t, client.Delete("https://x.example/1"))
	require.Equal(t, 1, inner.saved)
	require.Equal(t, 1, inner.deleted)
}

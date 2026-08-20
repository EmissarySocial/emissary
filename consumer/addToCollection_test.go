package consumer

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// mockClient is a streams.Client that resolves bare-string document IDs from an
// in-memory map. A URI that is not present in the map returns an error, which lets
// the tests exercise findRootContext's load-failure path.
type mockClient struct {
	documents mapof.Any
}

// SetRootClient implements the streams.Client interface. The stub ignores the root client.
func (client mockClient) SetRootClient(streams.Client) {}

// Load implements the streams.Client interface, resolving a URI from this stub's in-memory map
func (client mockClient) Load(uri string, _ ...any) (streams.Document, error) {

	if value, ok := client.documents[uri]; ok {
		return streams.NewDocument(value, streams.WithClient(client)), nil
	}

	return streams.NilDocument(), derp.Internal("mockClient.Load", "Unknown URI", uri)
}

// Save implements the streams.Client interface. The stub discards all writes.
func (client mockClient) Save(streams.Document) error { return nil }

// Delete implements the streams.Client interface. The stub discards all deletes.
func (client mockClient) Delete(string) error { return nil }

// newDocument builds a streams.Document from a raw value, wired to a client that can
// resolve the supplied ancestor documents by their bare-string IDs.
func newDocument(client mockClient, value any) streams.Document {
	return streams.NewDocument(value, streams.WithClient(client))
}

// TestFindRootContext verifies how the reply tree is walked, including the hop limit and load failures
func TestFindRootContext(t *testing.T) {

	// A document that carries its own `context` returns it immediately, without
	// consulting the reply tree or the client.
	t.Run("ContextAtRoot", func(t *testing.T) {

		document := newDocument(mockClient{}, mapof.Any{
			vocab.PropertyID:      "https://example.com/1",
			vocab.PropertyContext: "https://example.com/collections/abc",
		})

		context, err := findRootContext(document, 5)

		require.Nil(t, err)
		require.Equal(t, "https://example.com/collections/abc", context)
	})

	// A document with no context and no `inReplyTo` yields an empty context (and no
	// error): there is nowhere to walk to.
	t.Run("NoContextNoParent", func(t *testing.T) {

		document := newDocument(mockClient{}, mapof.Any{
			vocab.PropertyID: "https://example.com/1",
		})

		context, err := findRootContext(document, 5)

		require.Nil(t, err)
		require.Equal(t, "", context)
	})

	// When the parent is inlined (a map rather than a bare URL), the walk recurses
	// into it directly, with no client load.
	t.Run("InlinedParentHasContext", func(t *testing.T) {

		document := newDocument(mockClient{}, mapof.Any{
			vocab.PropertyID: "https://example.com/reply",
			vocab.PropertyInReplyTo: mapof.Any{
				vocab.PropertyID:      "https://example.com/parent",
				vocab.PropertyContext: "https://example.com/collections/xyz",
			},
		})

		context, err := findRootContext(document, 5)

		require.Nil(t, err)
		require.Equal(t, "https://example.com/collections/xyz", context)
	})

	// When the parent is a bare URL, it is loaded from the client and its context is
	// returned.
	t.Run("BareStringParentHasContext", func(t *testing.T) {

		client := mockClient{documents: mapof.Any{
			"https://example.com/parent": mapof.Any{
				vocab.PropertyID:      "https://example.com/parent",
				vocab.PropertyContext: "https://example.com/collections/xyz",
			},
		}}

		document := newDocument(client, mapof.Any{
			vocab.PropertyID:        "https://example.com/reply",
			vocab.PropertyInReplyTo: "https://example.com/parent",
		})

		context, err := findRootContext(document, 5)

		require.Nil(t, err)
		require.Equal(t, "https://example.com/collections/xyz", context)
	})

	// The walk climbs several bare-string ancestors until it finds the one that
	// carries a context.
	t.Run("WalksMultipleAncestors", func(t *testing.T) {

		client := mockClient{documents: mapof.Any{
			"https://example.com/2": mapof.Any{
				vocab.PropertyID:        "https://example.com/2",
				vocab.PropertyInReplyTo: "https://example.com/3",
			},
			"https://example.com/3": mapof.Any{
				vocab.PropertyID:      "https://example.com/3",
				vocab.PropertyContext: "https://example.com/collections/root",
			},
		}}

		document := newDocument(client, mapof.Any{
			vocab.PropertyID:        "https://example.com/1",
			vocab.PropertyInReplyTo: "https://example.com/2",
		})

		context, err := findRootContext(document, 5)

		require.Nil(t, err)
		require.Equal(t, "https://example.com/collections/root", context)
	})

	// A failure loading a bare-string ancestor is propagated so the caller can
	// requeue, rather than being swallowed into an empty (no-context) result.
	t.Run("LoadFailurePropagates", func(t *testing.T) {

		// The client's document map is empty, so loading the parent fails.
		document := newDocument(mockClient{}, mapof.Any{
			vocab.PropertyID:        "https://example.com/reply",
			vocab.PropertyInReplyTo: "https://example.com/missing-parent",
		})

		context, err := findRootContext(document, 5)

		require.NotNil(t, err)
		require.Equal(t, "", context)
	})

	// The depth counter bounds the walk. A chain of contextless ancestors deeper than
	// `count` stops and returns an empty context without erroring.
	t.Run("DepthLimitStopsWalk", func(t *testing.T) {

		// A chain of contextless documents, each replying to the next.
		client := mockClient{documents: mapof.Any{
			"https://example.com/2": mapof.Any{vocab.PropertyID: "https://example.com/2", vocab.PropertyInReplyTo: "https://example.com/3"},
			"https://example.com/3": mapof.Any{vocab.PropertyID: "https://example.com/3", vocab.PropertyInReplyTo: "https://example.com/4"},
			"https://example.com/4": mapof.Any{vocab.PropertyID: "https://example.com/4", vocab.PropertyInReplyTo: "https://example.com/5"},
		}}

		document := newDocument(client, mapof.Any{
			vocab.PropertyID:        "https://example.com/1",
			vocab.PropertyInReplyTo: "https://example.com/2",
		})

		// A budget of 1 allows loading exactly one ancestor (doc/2), which still has no
		// context, so the walk stops at the limit.
		context, err := findRootContext(document, 1)

		require.Nil(t, err)
		require.Equal(t, "", context)
	})

	// With count == 0 the walk does not consult the reply tree at all: a root document
	// without its own context returns empty even though a parent exists.
	t.Run("ZeroCountDoesNotWalk", func(t *testing.T) {

		// If the walk were to load the parent, it would find a context; the zero budget
		// must prevent that.
		client := mockClient{documents: mapof.Any{
			"https://example.com/parent": mapof.Any{
				vocab.PropertyID:      "https://example.com/parent",
				vocab.PropertyContext: "https://example.com/collections/xyz",
			},
		}}

		document := newDocument(client, mapof.Any{
			vocab.PropertyID:        "https://example.com/reply",
			vocab.PropertyInReplyTo: "https://example.com/parent",
		})

		context, err := findRootContext(document, 0)

		require.Nil(t, err)
		require.Equal(t, "", context)
	})
}

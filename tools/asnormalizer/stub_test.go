package asnormalizer

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestStub_HoldsOnlyItsID confirms that the stub is an object whose only property is the URL it
// stands in for, marked no-store.
func TestStub_HoldsOnlyItsID(t *testing.T) {

	result := stub("https://example.com/note")

	require.Equal(t, map[string]any{vocab.PropertyID: "https://example.com/note"}, result.Value())
	require.Equal(t, cacheheader.DirectiveNoStore, result.HTTPHeader().Get(cacheheader.HeaderCacheControl))
}

// TestStub_GettersNeverLoad confirms that reading the stub, the way the layers above the normalizer
// do, never starts a load.
func TestStub_GettersNeverLoad(t *testing.T) {

	// BUG-212: a bare-string stub would load its own URL from every one of these getters.
	inner := newFakeInner(nil)
	result := stub("https://example.com/note").AddOptions(streams.WithClient(inner))

	require.Equal(t, "https://example.com/note", result.ID())
	require.Equal(t, vocab.Unknown, result.Type())
	require.False(t, result.IsActivity())
	require.Empty(t, result.ActorID())
	require.Empty(t, result.Name())
	require.Empty(t, result.AttributedTo().ID())
	require.Empty(t, result.PublicKey().PublicKeyPEM())
	require.True(t, result.UnwrapActivity().IsMap())

	require.Zero(t, inner.totalLoads())
}

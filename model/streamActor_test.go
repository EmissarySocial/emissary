package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestStreamActorContext_Security guards BUG-05. handler/activitypub_stream/stream.go attaches a
// `publicKey` to whatever StreamActor.JSONLD returns, so this document MUST declare the security
// vocabulary -- a strict JSON-LD consumer drops undeclared terms, and a dropped `publicKey` means
// HTTP Signature verification against a Stream actor fails.
func TestStreamActorContext_Security(t *testing.T) {

	actor := StreamActor{SocialRole: vocab.ActorTypeService}

	stream := NewStream()
	stream.URL = "https://example.com/123"

	encoded, err := json.Marshal(actor.JSONLD(&stream))
	require.Nil(t, err)

	var decoded map[string]any
	require.Nil(t, json.Unmarshal(encoded, &decoded))

	// @context must be a list, not the bare ActivityStreams string
	contextList, ok := decoded[vocab.AtContext].([]any)
	require.True(t, ok, "@context must be a JSON array")
	require.Contains(t, contextList, vocab.ContextTypeActivityStreams)
	require.Contains(t, contextList, vocab.ContextTypeSecurity)
}

// TestStreamActorJSONLD_Empty confirms that an undefined Actor produces no document at all, which
// is what lets the handler skip the key lookup entirely.
func TestStreamActorJSONLD_Empty(t *testing.T) {

	actor := StreamActor{}
	stream := NewStream()

	require.True(t, actor.IsNil())
	require.Empty(t, actor.JSONLD(&stream))
}

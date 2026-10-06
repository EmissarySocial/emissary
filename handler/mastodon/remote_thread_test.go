package mastodon

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

// TestDocumentAuthorURL covers attributedTo, the actor fallback, and neither.
func TestDocumentAuthorURL(t *testing.T) {

	byAttribution := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/1", "attributedTo": "https://example.com/users/ben", "actor": "https://example.com/users/other"})
	require.Equal(t, "https://example.com/users/ben", documentAuthorURL(byAttribution))

	byActor := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/2", "actor": "https://example.com/users/amy"})
	require.Equal(t, "https://example.com/users/amy", documentAuthorURL(byActor))

	neither := streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/3"})
	require.Equal(t, "", documentAuthorURL(neither))
}

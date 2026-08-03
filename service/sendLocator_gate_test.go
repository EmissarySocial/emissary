package service

import (
	"slices"
	"testing"

	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// blockList returns a deliveryBlocked func that blocks exactly the given actor URLs.
func blockList(blocked ...string) func(data.Session, primitive.ObjectID, string) bool {
	return func(_ data.Session, _ primitive.ObjectID, actorURL string) bool {
		return slices.Contains(blocked, actorURL)
	}
}

// TestSendLocator_RecipientPreloadGate confirms a blocked URI yields no recipients WITHOUT
// fetching anything: the locator's client services are nil here, so any load attempt would panic.
func TestSendLocator_RecipientPreloadGate(t *testing.T) {

	locator := SendLocator{deliveryBlocked: blockList("https://evil.example/@spammer")}

	seq, err := locator.Recipient("https://evil.example/@spammer")
	require.NoError(t, err)
	require.Empty(t, slices.Collect(seq))
}

// TestSendLocator_RecipientPublicMarker confirms the Public audience marker (all three AS2
// forms) yields zero recipients WITHOUT fetching anything and WITHOUT consulting the rule
// filter: the locator's client services and deliveryBlocked func are nil here, so a
// fall-through to either would panic the test.
func TestSendLocator_RecipientPublicMarker(t *testing.T) {

	locator := SendLocator{}

	for _, uri := range []string{
		vocab.NamespaceActivityStreamsPublic, // "https://www.w3.org/ns/activitystreams#Public"
		vocab.NamespaceASPublic,              // "as:Public"
		vocab.NamespacePublic,                // "Public"
	} {
		seq, err := locator.Recipient(uri)
		require.NoError(t, err, "uri %q", uri)
		require.Empty(t, slices.Collect(seq), "uri %q", uri)
	}
}

// TestSendLocator_CollectionFilter confirms blocked members are stripped from an addressed
// collection while the rest still resolve to their inboxes (R4).
func TestSendLocator_CollectionFilter(t *testing.T) {

	locator := SendLocator{deliveryBlocked: blockList("https://blocked.example/@bob")}

	collection := streams.NewDocument(map[string]any{
		"type": "Collection",
		"items": []any{
			map[string]any{"type": "Person", "id": "https://ok.example/@alice", "inbox": "https://ok.example/@alice/inbox"},
			map[string]any{"type": "Person", "id": "https://blocked.example/@bob", "inbox": "https://blocked.example/@bob/inbox"},
		},
	})

	seq, err := locator.resolveCollection(collection)
	require.NoError(t, err)
	require.Equal(t, []string{"https://ok.example/@alice/inbox"}, slices.Collect(seq))
}

// TestSendLocator_BoundToSender confirms the rules binding: a local User actor binds their own
// UserID; every other actor URL binds the admin tier (NilObjectID).
func TestSendLocator_BoundToSender(t *testing.T) {

	userID := primitive.NewObjectID()
	locator := SendLocator{host: "https://example.com"}

	bound := locator.BoundToSender("https://example.com/@" + userID.Hex())
	require.Equal(t, userID, bound.ruleUserID)

	remote := locator.BoundToSender("https://remote.example/@someone")
	require.True(t, remote.ruleUserID.IsZero())
}

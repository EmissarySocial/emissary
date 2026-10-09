package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestOutboxMessage verifies that every OutboxMessage property round-trips through the schema
func TestOutboxMessage(t *testing.T) {

	s := schema.New(OutboxMessageSchema())
	response := NewOutboxMessage()

	tests := []tableTestItem{
		{"outboxMessageId", "000000000000000000000001", nil},
		{"actorType", "User", nil},
		{"actorId", "000000000000000000000001", nil},
		{"activityType", "Create", nil},
		{"objectId", "https://john.connor.mil", nil},
		{"permissions.0", "086753090867530908675309", nil},
		{"permissions.1", "086753090867530908675309", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}

// TestOutboxMessage_ActivityPubURL verifies which URL identifies a message to ActivityPub
func TestOutboxMessage_ActivityPubURL(t *testing.T) {

	messageID, err := primitive.ObjectIDFromHex("687aeeef6c699789c9ce623a")
	require.Nil(t, err)

	testCases := []struct {
		name        string
		actorURL    string
		activityURL string
		expected    string
	}{
		// A stored, canonical ActivityURL always wins
		{"canonical activity URL", "https://x.social/@bob", "https://x.social/@bob/pub/liked/1", "https://x.social/@bob/pub/liked/1"},
		{"canonical URL with no actor", "", "https://x.social/@bob/pub/liked/1", "https://x.social/@bob/pub/liked/1"},

		// Otherwise the URL is minted from the Actor
		{"minted from user actor", "https://x.social/@bob", "", "https://x.social/@bob/pub/outbox/687aeeef6c699789c9ce623a"},
		{"minted from stream actor", "https://x.social/123", "", "https://x.social/123/pub/outbox/687aeeef6c699789c9ce623a"},

		// RULE: never mint a relative ID -- the whole point of BUG-146
		{"no actor and no activity", "", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			message := NewOutboxMessage()
			message.OutboxMessageID = messageID
			message.ActorURL = testCase.actorURL
			message.ActivityURL = testCase.activityURL

			require.Equal(t, testCase.expected, message.ActivityPubURL())
		})
	}
}

// TestOutboxMessage_GetJSONLD verifies that a message renders as an ActivityStreams activity
func TestOutboxMessage_GetJSONLD(t *testing.T) {

	message := NewOutboxMessage()
	message.ActorURL = "https://x.social/@bob"
	message.ActivityType = vocab.ActivityTypeCreate
	message.ObjectID = "https://x.social/123"
	message.Permissions = NewAnonymousPermissions()

	result := message.GetJSONLD()

	require.Equal(t, vocab.ContextTypeActivityStreams, result[vocab.AtContext])
	require.Equal(t, "https://x.social/@bob", result.GetString(vocab.PropertyActor))
	require.Equal(t, vocab.ActivityTypeCreate, result.GetString(vocab.PropertyType))
	require.Equal(t, "https://x.social/123", result.GetString(vocab.PropertyObject))
	require.Equal(t, message.ActivityPubURL(), result.GetString(vocab.PropertyID))
	require.Equal(t, []string{vocab.NamespacePublic}, result[vocab.PropertyTo])
}

// TestOutboxMessage_GetJSONLD_NoActor verifies that a message with no Actor renders no `id` at all,
// so the collection builder can omit it instead of publishing a relative path (BUG-146)
func TestOutboxMessage_GetJSONLD_NoActor(t *testing.T) {

	message := NewOutboxMessage()
	message.ActivityType = vocab.ActivityTypeLike

	result := message.GetJSONLD()

	require.Empty(t, result.GetString(vocab.PropertyID))
	require.Empty(t, result.GetString(vocab.PropertyActor))
}

// TestOutboxMessage_GetJSONLD_Private verifies that a non-anonymous message is not addressed to the public
func TestOutboxMessage_GetJSONLD_Private(t *testing.T) {

	message := NewOutboxMessage()
	message.ActorURL = "https://x.social/@bob"
	message.Permissions = NewPermissions()
	message.Permissions = append(message.Permissions, primitive.NewObjectID())

	result := message.GetJSONLD()

	require.Equal(t, []string{}, result[vocab.PropertyTo])
}

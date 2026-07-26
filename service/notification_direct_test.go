package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testRecipient builds a User whose ActivityPubURL is a known, stable value.
func testRecipient() *model.User {
	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.ProfileURL = "https://local.example/@me"
	return &user
}

// TestIsDirectMessage pins the DIRECT classification.  Both halves of the test are load-bearing:
// non-public alone would also match a followers-only post (a timeline post, not a conversation),
// and addressing alone would match every public post that names the user.
func TestIsDirectMessage(t *testing.T) {

	user := testRecipient()
	me := user.ActivityPubURL()
	require.NotEmpty(t, me, "test setup: recipient must have an ActivityPub URL")

	table := []struct {
		name     string
		activity mapof.Any
		expected bool
	}{
		{
			name: "private message addressed to me",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyTo:   []any{me, "https://remote.example/@friend"},
			},
			expected: true,
		},
		{
			name: "private message that addresses me only on the wrapped object",
			activity: mapof.Any{
				vocab.PropertyType:   vocab.ActivityTypeCreate,
				vocab.PropertyObject: mapof.Any{vocab.PropertyTo: []any{me}},
			},
			expected: true,
		},
		{
			name: "private message addressed via cc",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyCC:   []any{me},
			},
			expected: true,
		},
		{
			name: "followers-only post is NOT direct -- it names the followers collection, not me",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyTo:   []any{"https://remote.example/@alice/followers"},
			},
			expected: false,
		},
		{
			name: "public post that names me is NOT direct",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyTo:   []any{vocab.NamespaceActivityStreamsPublic},
				vocab.PropertyCC:   []any{me},
			},
			expected: false,
		},
		{
			name: "quiet-public post that names me is NOT direct",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyTo:   []any{me},
				vocab.PropertyCC:   []any{vocab.NamespaceActivityStreamsPublic},
			},
			expected: false,
		},
		{
			name: "private message to somebody else is not mine",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
				vocab.PropertyTo:   []any{"https://remote.example/@somebody"},
			},
			expected: false,
		},
		{
			name: "no addressing at all is not a direct message",
			activity: mapof.Any{
				vocab.PropertyType: vocab.ActivityTypeCreate,
			},
			expected: false,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, isDirectMessage(streams.NewDocument(test.activity), user))
		})
	}
}

// TestIsDirectMessage_NoActorURL covers the degenerate case: a User with no ActivityPub URL must
// never match, or an object with an empty recipient string would classify as a DM for them.
func TestIsDirectMessage_NoActorURL(t *testing.T) {

	user := model.NewUser()

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyTo:   []any{""},
	})

	require.False(t, isDirectMessage(activity, &user))
}

// TestMessageCodec pins the DIRECT Subtype vocabulary.  Getting MLS wrong means base64 ciphertext
// is snapshotted into ObjectSummary and pushed to the recipient's lock screen.
func TestMessageCodec(t *testing.T) {

	table := []struct {
		name     string
		object   mapof.Any
		expected string
	}{
		{
			name:     "MLS ciphertext",
			object:   mapof.Any{vocab.PropertyMediaType: vocab.MediaTypeMLS},
			expected: model.NotificationSubtypeMLS,
		},
		{
			name:     "html content",
			object:   mapof.Any{vocab.PropertyMediaType: "text/html"},
			expected: model.NotificationSubtypePlaintext,
		},
		{
			name:     "no media type",
			object:   mapof.Any{vocab.PropertyContent: "Hello."},
			expected: model.NotificationSubtypePlaintext,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, messageCodec(streams.NewDocument(test.object)))
		})
	}
}

package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestMessageForTopic pins each topic's event-name and payload conventions
// (mirroring realtime/message.go) so a constructor change that would silently
// break subscribed templates is caught here.
func TestMessageForTopic(t *testing.T) {

	objectID := primitive.NewObjectID()

	// test asserts one topic's conversion: the expected event name and payload,
	// with the ObjectID and topic checked identically for every case.
	test := func(topic int, wantEvent string, wantData string) {
		t.Helper()

		message, err := messageForTopic(topic, objectID, "payload")
		require.NoError(t, err, "topic %d", topic)
		require.Equal(t, objectID, message.ObjectID, "topic %d", topic)
		require.Equal(t, topic, message.Topic, "topic %d", topic)
		require.Equal(t, wantEvent, message.Event, "topic %d", topic)
		require.Equal(t, wantData, message.Data, "topic %d", topic)
	}

	hex := objectID.Hex()

	// Object-scoped topics publish under the object's hex ID with a fixed payload
	test(realtime.TopicUpdated, hex, "updated")
	test(realtime.TopicChildUpdated, hex, "child updated")
	test(realtime.TopicNewReplies, hex, "new replies")
	test(realtime.TopicImportProgress, hex, "import progress")
	test(realtime.TopicFollowingUpdated, hex, "following updated")

	// Default-event topics publish as "message"; inbox topics carry the data argument
	test(realtime.TopicInboxActivity, "", "payload")
	test(realtime.TopicInboxActivity_DirectMessage, "", "payload")
	test(realtime.TopicInboxActivity_DirectMessage_MLS, "", "payload")
	test(realtime.TopicNotification, "", "notification")
}

// TestMessageForTopic_Unrecognized verifies that a filter-only or unknown topic errors instead of publishing
func TestMessageForTopic_Unrecognized(t *testing.T) {

	// TopicAll is a subscription filter, not a publishable topic — and any
	// unknown value must error rather than deliver a zero message.
	_, err := messageForTopic(realtime.TopicAll, primitive.NewObjectID(), "")
	require.Error(t, err)

	_, err = messageForTopic(999, primitive.NewObjectID(), "")
	require.Error(t, err)
}

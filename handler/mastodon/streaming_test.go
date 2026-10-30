package mastodon

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamingToken confirms the subprotocol wins over the query, which wins over the header
func TestStreamingToken(t *testing.T) {

	request := httptest.NewRequest("GET", "/api/v1/streaming?access_token=query", nil)
	request.Header.Set("Sec-WebSocket-Protocol", "protocol")
	request.Header.Set("Authorization", "Bearer header")
	require.Equal(t, "protocol", streamingToken(request))

	request.Header.Del("Sec-WebSocket-Protocol")
	require.Equal(t, "query", streamingToken(request))

	request = httptest.NewRequest("GET", "/api/v1/streaming", nil)
	request.Header.Set("Authorization", "Bearer header")
	require.Equal(t, "header", streamingToken(request))

	require.Equal(t, "", streamingToken(httptest.NewRequest("GET", "/api/v1/streaming", nil)))
}

// TestNewStreamingEvent confirms the payload is a JSON string that decodes to the entity
func TestNewStreamingEvent(t *testing.T) {

	event, err := newStreamingEvent(streamUser, "update", map[string]string{"id": "123"})
	require.NoError(t, err)

	encoded, err := json.Marshal(event)
	require.NoError(t, err)

	var frame map[string]any
	require.NoError(t, json.Unmarshal(encoded, &frame))
	require.Equal(t, []any{"user"}, frame["stream"])
	require.Equal(t, "update", frame["event"])

	payload, isString := frame["payload"].(string)
	require.True(t, isString, "the payload is a string, not a nested object")

	var entity map[string]string
	require.NoError(t, json.Unmarshal([]byte(payload), &entity))
	require.Equal(t, "123", entity["id"])
}

// TestStreamingSubscriptions confirms subscribe and unsubscribe
func TestStreamingSubscriptions(t *testing.T) {

	subs := newStreamingSubscriptions()
	require.False(t, subs.has(streamUser))

	subs.set(streamUser, true)
	require.True(t, subs.has(streamUser))
	require.False(t, subs.has(streamUserNotification))

	subs.set(streamUser, false)
	require.False(t, subs.has(streamUser))
}

// TestStreamingEventsForMessage_Skips confirms unsubscribed or ID-less messages never reach the database
func TestStreamingEventsForMessage_Skips(t *testing.T) {

	auth := model.Authorization{UserID: primitive.NewObjectID()}
	subs := newStreamingSubscriptions()
	subs.set(streamUser, true)

	// withRecord gives a message a record ID
	withRecord := func(message realtime.Message) realtime.Message {
		message.RecordID = primitive.NewObjectID()
		return message
	}

	// No record ID: nothing to load
	require.Empty(t, streamingEventsForMessage(nil, nil, auth, realtime.NewMessage_NewsItem(auth.UserID), subs))

	// Not subscribed: nothing to send
	none := newStreamingSubscriptions()
	require.Empty(t, streamingEventsForMessage(nil, nil, auth, withRecord(realtime.NewMessage_NewsItem(auth.UserID)), none))
	require.Empty(t, streamingEventsForMessage(nil, nil, auth, withRecord(realtime.NewMessage_Notification(auth.UserID)), none))

	// A sent direct message only goes to the "direct" stream
	require.Empty(t, streamingEventsForMessage(nil, nil, auth, withRecord(realtime.NewMessage_DirectMessageSent(auth.UserID)), subs))

	// Topics the stream does not carry are skipped
	require.Empty(t, streamingEventsForMessage(nil, nil, auth, withRecord(realtime.NewMessage_Updated(auth.UserID)), subs))
}

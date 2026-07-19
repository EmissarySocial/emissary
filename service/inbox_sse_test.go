package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// sseTopics runs sendSSEUpdate on the provided activity and returns the topics it emitted, in order.
func sseTopics(t *testing.T, activity model.InboxActivity) []int {
	t.Helper()

	channel := make(chan realtime.Message, 8)
	service := Inbox{sseUpdateChannel: channel}

	service.sendSSEUpdate(&activity)
	close(channel)

	result := make([]int, 0, len(channel))
	for message := range channel {
		result = append(result, message.Topic)
	}

	return result
}

// TestInbox_SendSSEUpdate_Topics confirms the topic cascade: every stored activity nudges the
// generic Inbox topic, non-public activities add the DirectMessage topic, and MLS messages add
// the MLS topic on top of both.
func TestInbox_SendSSEUpdate_Topics(t *testing.T) {

	public := model.InboxActivity{UserID: primitive.NewObjectID(), IsPublic: true}
	require.Equal(t, []int{realtime.TopicInboxActivity}, sseTopics(t, public))

	directMessage := model.InboxActivity{UserID: primitive.NewObjectID(), IsPublic: false}
	require.Equal(t,
		[]int{realtime.TopicInboxActivity, realtime.TopicInboxActivity_DirectMessage},
		sseTopics(t, directMessage))

	mls := model.InboxActivity{UserID: primitive.NewObjectID(), IsPublic: false, MediaType: vocab.MediaTypeMLS}
	require.Equal(t,
		[]int{realtime.TopicInboxActivity, realtime.TopicInboxActivity_DirectMessage, realtime.TopicInboxActivity_DirectMessage_MLS},
		sseTopics(t, mls))
}

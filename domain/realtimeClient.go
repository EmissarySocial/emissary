package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RealtimeClient represents a single SSE connection that has subscribed to updates for a particular stream/view combination.
type RealtimeClient struct {
	ClientID     primitive.ObjectID      // Unique Identifier of this RealtimeClient.
	HTTPRequest  *HTTPRequest            // HTTP Request that initiated the client
	StreamID     primitive.ObjectID      // Stream.Token of current stream being watched.
	WriteChannel chan primitive.ObjectID // Channel for writing responses to this client.
}

// NewRealtimeClient initializes a new realtime client.
func NewRealtimeClient(httpRequest *HTTPRequest, streamID primitive.ObjectID) *RealtimeClient {

	return &RealtimeClient{
		ClientID:     primitive.NewObjectID(),
		HTTPRequest:  httpRequest,
		StreamID:     streamID,
		WriteChannel: make(chan primitive.ObjectID),
	}
}

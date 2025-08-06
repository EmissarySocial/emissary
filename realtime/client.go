package realtime

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Client represents a single SSE connection that has subscribed to updates for a particular stream/view combination.
type Client struct {
	ClientID     primitive.ObjectID      // Unique Identifier of this Client.
	Request      *http.Request           // HTTP Request that initiated the client
	StreamID     primitive.ObjectID      // Stream.Token of current stream being watched.
	WriteChannel chan primitive.ObjectID // Channel for writing responses to this client.
}

// NewClient initializes a new realtime client.
func NewClient(request *http.Request, streamID primitive.ObjectID) *Client {

	return &Client{
		ClientID:     primitive.NewObjectID(),
		Request:      request,
		StreamID:     streamID,
		WriteChannel: make(chan primitive.ObjectID),
	}
}

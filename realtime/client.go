package realtime

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// writeChannelBuffer bounds how many undelivered nudges the broker holds per client
// before it starts dropping them.  Messages are "something changed, refetch" signals,
// so a full buffer just means this client is briefly behind; the next nudge (or a page
// load) brings it current.  The buffer lets the broker use a non-blocking send so one
// slow client can never stall delivery to the others.
const writeChannelBuffer = 16

// Client represents a single SSE connection that has subscribed to updates for a particular stream/view combination.
type Client struct {
	ClientID     primitive.ObjectID // Unique Identifier of this Client.
	Request      *http.Request      // HTTP Request that initiated the client
	StreamID     primitive.ObjectID // Stream.Token of current stream being watched.
	Topic        int
	WriteChannel chan Message // Channel for writing responses to this client.
}

// NewClient initializes a new realtime client.
func NewClient(request *http.Request, streamID primitive.ObjectID, topic int) *Client {

	return &Client{
		ClientID:     primitive.NewObjectID(),
		Request:      request,
		StreamID:     streamID,
		Topic:        topic,
		WriteChannel: make(chan Message, writeChannelBuffer),
	}
}

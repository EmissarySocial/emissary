package domain

import (
	"github.com/EmissarySocial/emissary/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RealtimeBroker is a singleton. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
type RealtimeBroker struct {

	// map of realtime clients
	clients map[primitive.ObjectID]*RealtimeClient

	// map of streams being watched.
	streams map[primitive.ObjectID]map[primitive.ObjectID]*RealtimeClient

	// Channel that streams are pushed into when they change.
	streamUpdates chan model.Stream

	// Channel into which new clients can be pushed
	AddClient chan *RealtimeClient

	// Channel into which disconnected clients should be pushed
	RemoveClient chan *RealtimeClient

	// Channel into which the broker should be closed
	close chan bool
}

// NewRealtimeBroker generates a new stream broker
func NewRealtimeBroker(factory *Factory, updates chan model.Stream) RealtimeBroker {

	result := RealtimeBroker{
		clients:       make(map[primitive.ObjectID]*RealtimeClient),
		streams:       make(map[primitive.ObjectID]map[primitive.ObjectID]*RealtimeClient),
		streamUpdates: updates,
		AddClient:     make(chan *RealtimeClient),
		RemoveClient:  make(chan *RealtimeClient),
		close:         make(chan bool),
	}

	go result.listen(factory)

	return result
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh
func (b *RealtimeBroker) Refresh() {
	// No operation is required right now.  Included for API consistency.
}

// Stop closes the broker
func (b *RealtimeBroker) Close() {
	close(b.close)
}

/*******************************************
 * LISTEN/NOTIFY METHODS
 *******************************************/

// Listen handles the addition & removal of clients, as well as
// the broadcasting of messages out to clients that are currently attached.
// It is intended to be run in its own goroutine.
func (b *RealtimeBroker) listen(factory *Factory) {

	for {

		// Block until we receive from one of the
		// three following channels.
		select {

		case client := <-b.AddClient:

			if _, ok := b.streams[client.StreamID]; !ok {
				b.streams[client.StreamID] = make(map[primitive.ObjectID]*RealtimeClient)
			}

			b.streams[client.StreamID][client.ClientID] = client
			b.clients[client.ClientID] = client

			// log.Println("Added new client")

		case client := <-b.RemoveClient:

			delete(b.clients, client.ClientID)
			delete(b.streams[client.StreamID], client.ClientID)

			if len(b.streams[client.StreamID]) == 0 {
				delete(b.streams, client.StreamID)
			}

			close(client.WriteChannel)

			// log.Println("Removed client")

		case stream := <-b.streamUpdates:

			// Send an update to every client that has subscribed to this stream
			go b.notify(stream.StreamID)

			// Try to send updates to every client that has subscribed to this stream's parent
			if stream.HasParent() {
				go b.notify(stream.ParentID)
			}

		case <-b.close:
			return
		}
	}
}

// notify sends updates for every client that is watching a given stream
func (b *RealtimeBroker) notify(streamID primitive.ObjectID) {

	for _, client := range b.streams[streamID] {
		client.WriteChannel <- streamID
	}

}

package domain

import (
	"fmt"

	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RealtimeBroker is a singleton. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
type RealtimeBroker struct {

	// map of realtime clients
	clients map[primitive.ObjectID]*RealtimeClient

	// map of streams being watched.
	streams map[string]map[primitive.ObjectID]*RealtimeClient

	// Channel that streams are pushed into when they change.
	streamUpdates chan model.Stream

	// Channel into which new clients can be pushed
	AddClient chan *RealtimeClient

	// Channel into which disconnected clients should be pushed
	RemoveClient chan *RealtimeClient
}

// NewRealtimeBroker generates a new stream broker
func NewRealtimeBroker(factory *Factory, updates chan model.Stream) *RealtimeBroker {

	result := &RealtimeBroker{
		clients:       make(map[primitive.ObjectID]*RealtimeClient),
		streams:       make(map[string]map[primitive.ObjectID]*RealtimeClient),
		streamUpdates: updates,
		AddClient:     make(chan *RealtimeClient),
		RemoveClient:  make(chan *RealtimeClient),
	}

	go result.Listen(factory)

	return result
}

// Listen handles the addition & removal of clients, as well as
// the broadcasting of messages out to clients that are currently attached.
// It is intended to be run in its own goroutine.
func (b *RealtimeBroker) Listen(factory *Factory) {

	//streamService := factory.Stream()

	for {

		// Block until we receive from one of the
		// three following channels.
		select {

		case client := <-b.AddClient:

			if _, ok := b.streams[client.Token]; !ok {
				b.streams[client.Token] = make(map[primitive.ObjectID]*RealtimeClient)
			}

			b.streams[client.Token][client.ClientID] = client
			b.clients[client.ClientID] = client

			// log.Println("Added new client")

		case client := <-b.RemoveClient:

			delete(b.clients, client.ClientID)
			delete(b.streams[client.Token], client.ClientID)

			if len(b.streams[client.Token]) == 0 {
				delete(b.streams, client.Token)
			}

			close(client.WriteChannel)

			// log.Println("Removed client")

		case stream := <-b.streamUpdates:

			// Send an update to every client that has subscribed to this stream
			b.notify(stream)

			/***********
			Halt notification on parents for now, because of null pointer errors..

			// Try to send updates to every client that has subscribed to this stream's parent
			if stream.HasParent() {

				parent := model.NewStream()
				parent.Label = "Populated..."

				if err := streamService.LoadParent(&stream, &parent); err != nil {
					derp.Report(derp.Wrap(err, "ghost.domain.Realtimebroker", "Error loading parent stream to update parent's subscribers."))
					continue
				}

				b.notify(parent)
			}
			*/
		}
	}
}

// notify sends updates for every client that is watching a given stream
func (b *RealtimeBroker) notify(stream model.Stream) {

	fmt.Println(stream.Token)
	for key := range b.streams {
		fmt.Println(key)
	}

	for _, client := range b.streams[stream.Token] {
		fmt.Println("notifying: " + client.ClientID.Hex())
		client.WriteChannel <- stream.StreamID
	}
}

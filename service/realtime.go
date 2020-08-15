package service

import (
	"log"

	"github.com/benpate/derp"
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

	// Channels that updates are pushed into.
	streamUpdates   chan model.Stream
	templateUpdates chan model.Template

	// Create a map of clients, the keys of the map are the channels
	// over which we can push messages to attached clients.  (The values
	// are just booleans and are meaningless.)
	// clients map[chan string]bool

	// Channel into which new clients can be pushed
	AddClient chan *RealtimeClient

	// Channel into which disconnected clients should be pushed
	RemoveClient chan *RealtimeClient
}

type RealtimeClient struct {
	ClientID     primitive.ObjectID // Unique Identifier of this RealtimeClient.
	Token        string             // Stream.Token of current stream being watched.
	View         string             // Stream.View of the current stream/view being watched.
	WriteChannel chan string        // Channel for writing responses to this client.
}

// NewRealtimeBroker generates a new stream broker
func NewRealtimeBroker(factory Factory) *RealtimeBroker {

	result := &RealtimeBroker{
		clients:         make(map[primitive.ObjectID]*RealtimeClient),
		streams:         make(map[string]map[primitive.ObjectID]*RealtimeClient),
		streamUpdates:   factory.StreamWatcher(),
		templateUpdates: make(chan model.Template),
		AddClient:       make(chan *RealtimeClient),
		RemoveClient:    make(chan *RealtimeClient),
	}

	go result.Listen(factory)

	return result
}

// NewRealtimeClient initializes a new realtime client.
func NewRealtimeClient(token string, view string) *RealtimeClient {

	return &RealtimeClient{
		ClientID:     primitive.NewObjectID(),
		Token:        token,
		View:         view,
		WriteChannel: make(chan string),
	}
}

// Listen handles
// the addition & removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
//
func (b *RealtimeBroker) Listen(factory Factory) {

	// Get the stream service
	streamService := factory.Stream()

	// Loop endlessly
	//
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

			log.Println("Added new client")

		case client := <-b.RemoveClient:

			delete(b.clients, client.ClientID)
			delete(b.streams[client.Token], client.ClientID)

			if len(b.streams[client.Token]) == 0 {
				delete(b.streams, client.Token)
			}

			close(client.WriteChannel)

			log.Println("Removed client")

		case stream := <-b.streamUpdates:

			for _, client := range b.streams[stream.Token] {

				if html, err := streamService.Render(&stream, client.View); err == nil {
					client.WriteChannel <- html
				}
			}

		case template := <-b.templateUpdates:

			go func() {

				iterator, err := streamService.ListByTemplate(template.TemplateID)

				if err != nil {
					derp.Report(derp.Wrap(err, "ghost.service.Realtime", "Error Listing Streams for Template", template))
					return
				}

				var stream model.Stream

				for iterator.Next(&stream) {
					b.streamUpdates <- stream
				}
			}()
		}
	}
}

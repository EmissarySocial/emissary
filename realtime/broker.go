package realtime

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// realtime.Broker is a singleton. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
//
// TODO: MEDIUM: Should the realtime broker be a service?
// Is there a reason to have multiple instances of the realtime broker, or should it be a GLOBAL service?
type Broker struct {

	// map of realtime clients
	clients map[primitive.ObjectID]*Client

	// map of streams being watched.
	streams map[primitive.ObjectID]map[primitive.ObjectID]*Client

	// Channel that users/streams are pushed into when they change.
	updateChannel chan Message

	// Channel into which new clients can be pushed
	AddClient chan *Client

	// Channel into which disconnected clients should be pushed
	RemoveClient chan *Client

	// Channel into which the broker should be closed
	close chan bool
}

// NewBroker generates a new stream broker
func NewBroker(updateChannel chan Message) Broker {

	result := Broker{
		clients:       make(map[primitive.ObjectID]*Client),
		streams:       make(map[primitive.ObjectID]map[primitive.ObjectID]*Client),
		updateChannel: updateChannel,

		AddClient:    make(chan *Client),
		RemoveClient: make(chan *Client),
		close:        make(chan bool),
	}

	go result.listen()

	return result
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates values in the broker that may have changed with the last config update.
func (b *Broker) Refresh() {
}

// Stop closes the broker
func (b *Broker) Close() {
	close(b.close)
}

/******************************************
 * Listen/Modify Methods
 ******************************************/

// Listen handles the addition & removal of clients, as well as
// the broadcasting of messages out to clients that are currently attached.
// It is intended to be run in its own goroutine.
func (b *Broker) listen() {

	for {

		// Block until we receive from one of the
		// three following channels.
		select {

		case client := <-b.AddClient:

			if _, ok := b.streams[client.StreamID]; !ok {
				b.streams[client.StreamID] = make(map[primitive.ObjectID]*Client)
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

		case message := <-b.updateChannel:

			go b.notifySSE(message)

		case <-b.close:
			return
		}
	}
}

// notifySSE sends updates for every SEE client that is watching a given stream
func (b *Broker) notifySSE(message Message) {

	// If the object is empty, then there's nothing to do
	if message.ObjectID.IsZero() {
		return
	}

	// Delay before sending updates on new replies
	// (wait for new items to settle in the database)
	if message.Topic == TopicNewReplies {
		time.Sleep(2 * time.Second)
	}

	// Send realtime messages to SSE clients
	for _, client := range b.streams[message.ObjectID] {
		if (client.Topic == TopicAll) || (client.Topic == message.Topic) {
			client.WriteChannel <- message.ObjectID
		}
	}
}

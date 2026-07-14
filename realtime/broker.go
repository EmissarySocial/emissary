package realtime

import (
	"sync"
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

	// mutex guards the clients and objects maps.  notifySSE reads them from its own
	// goroutine (one per message) while listen() mutates them on connect/disconnect;
	// without this lock that concurrent map access is a data race.
	mutex sync.RWMutex

	// map of realtime clients
	clients map[primitive.ObjectID]*Client

	// map of streams being watched.
	objects map[primitive.ObjectID]map[primitive.ObjectID]*Client

	// Channel that users/streams are pushed into when they change.
	updateChannel chan Message

	// Channel into which new clients can be pushed
	AddClient chan *Client

	// Channel into which disconnected clients should be pushed
	RemoveClient chan *Client

	// Channel into which the broker should be closed
	close chan bool
}

// NewBroker generates a new stream broker.  It returns a pointer: the Broker owns a
// mutex and a background goroutine, so it must never be copied.
func NewBroker(updateChannel chan Message) *Broker {

	result := &Broker{
		clients:       make(map[primitive.ObjectID]*Client),
		objects:       make(map[primitive.ObjectID]map[primitive.ObjectID]*Client),
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

		select {

		case client := <-b.AddClient:

			b.mutex.Lock()

			if _, ok := b.objects[client.StreamID]; !ok {
				b.objects[client.StreamID] = make(map[primitive.ObjectID]*Client)
			}

			b.objects[client.StreamID][client.ClientID] = client
			b.clients[client.ClientID] = client

			b.mutex.Unlock()

		case client := <-b.RemoveClient:

			b.mutex.Lock()

			delete(b.clients, client.ClientID)
			delete(b.objects[client.StreamID], client.ClientID)

			if len(b.objects[client.StreamID]) == 0 {
				delete(b.objects, client.StreamID)
			}

			b.mutex.Unlock()

			// RULE: Do NOT close(client.WriteChannel) here.  notifySSE runs in its own
			// goroutine and may still hold this client in a delivery snapshot; closing
			// while it sends would panic ("send on closed channel").  The client's handler
			// exits on its own request context (see handler.serverSentEvent), and the
			// channel is garbage-collected once the client is no longer referenced.

		case message := <-b.updateChannel:

			// Deliver in a separate goroutine so a slow client (or the NewReplies delay
			// inside notifySSE) never blocks this connect/disconnect loop.
			go b.notifySSE(message)

		case <-b.close:
			return
		}
	}
}

// notifySSE sends a message to every SSE client watching the message's object on a matching topic.
func (b *Broker) notifySSE(message Message) {

	// RULE: Delay before sending updates on "New Replies"
	// (hack to wait for new items to settle in the database)
	if message.Topic == TopicNewReplies {
		time.Sleep(2 * time.Second)
	}

	// Snapshot the matching clients under a read lock, then release it BEFORE sending.
	// Holding the lock across a channel send would serialize delivery against every
	// connect/disconnect and let one wedged client stall the rest.
	b.mutex.RLock()

	recipients := make([]*Client, 0, len(b.objects[message.ObjectID]))

	for _, client := range b.objects[message.ObjectID] {
		if (client.Topic == TopicAll) || (client.Topic == message.Topic) {
			recipients = append(recipients, client)
		}
	}

	b.mutex.RUnlock()

	// Deliver outside the lock with a non-blocking send.  A client whose buffer is full
	// simply misses this nudge — it's a "something changed, refetch" signal, so the next
	// event (or a page load) brings it current — rather than blocking everyone else.
	for _, client := range recipients {
		select {
		case client.WriteChannel <- message:
		default:
		}
	}
}

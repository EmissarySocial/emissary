package realtime

import (
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestBroker_ConcurrentAccess hammers the broker with concurrent connects,
// disconnects, and deliveries for the same object.  Its purpose is to be run
// under the race detector (`go test -race`): notifySSE reads the client maps
// from its own goroutine while listen() mutates them on connect/disconnect, so
// without the broker's mutex this is a data race.
func TestBroker_ConcurrentAccess(t *testing.T) {

	updateChannel := make(chan Message, 256)
	broker := NewBroker(updateChannel)
	defer broker.Close()

	userID := primitive.NewObjectID()

	var waitGroup sync.WaitGroup

	// Publisher: fire a stream of notification messages for userID, each of which
	// spawns a notifySSE goroutine that reads the client map.
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for index := 0; index < 500; index++ {
			updateChannel <- NewMessage_Notification(userID)
		}
	}()

	// Churn: several workers connect and immediately disconnect clients watching
	// userID, mutating the maps while the publisher's notifySSE goroutines read them.
	for worker := 0; worker < 8; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := 0; index < 200; index++ {
				client := NewClient(nil, userID, TopicNotification)
				broker.AddClient <- client
				broker.RemoveClient <- client
			}
		}()
	}

	waitGroup.Wait()

	// Let any in-flight notifySSE goroutines finish touching the maps before the
	// deferred Close() stops the broker.
	time.Sleep(50 * time.Millisecond)
}

// TestBroker_Delivery confirms a connected client on a matching topic actually
// receives a published message.  AddClient is sent first on an unbuffered channel,
// so listen() finishes registering the client before it can dequeue the message
// that spawns notifySSE — making the delivery deterministic.
func TestBroker_Delivery(t *testing.T) {

	updateChannel := make(chan Message)
	broker := NewBroker(updateChannel)
	defer broker.Close()

	userID := primitive.NewObjectID()
	client := NewClient(nil, userID, TopicNotification)

	broker.AddClient <- client
	updateChannel <- NewMessage_Notification(userID)

	select {
	case message := <-client.WriteChannel:
		if message.Topic != TopicNotification {
			t.Fatalf("expected TopicNotification (%d), got %d", TopicNotification, message.Topic)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE delivery")
	}
}

// TestBroker_Send confirms the direct-delivery entry point: Send delivers
// synchronously on the caller's goroutine, so a registered client's buffered
// WriteChannel holds the message as soon as Send returns.
func TestBroker_Send(t *testing.T) {

	broker := NewBroker(make(chan Message))
	defer broker.Close()

	userID := primitive.NewObjectID()
	client := NewClient(nil, userID, TopicNotification)

	// AddClient is unbuffered: once this send returns, listen() has registered the client.
	broker.AddClient <- client

	broker.Send(NewMessage_Notification(userID))

	select {
	case message := <-client.WriteChannel:
		if message.Topic != TopicNotification {
			t.Fatalf("expected TopicNotification (%d), got %d", TopicNotification, message.Topic)
		}
	default:
		t.Fatal("Send returned without delivering to the client's buffer")
	}
}

// TestBroker_TopicFilter confirms a client only receives messages for its own
// topic (or TopicAll), never an unrelated one.
func TestBroker_TopicFilter(t *testing.T) {

	updateChannel := make(chan Message)
	broker := NewBroker(updateChannel)
	defer broker.Close()

	userID := primitive.NewObjectID()

	// This client watches inbox activity, NOT notifications.
	client := NewClient(nil, userID, TopicInboxActivity)

	broker.AddClient <- client
	updateChannel <- NewMessage_Notification(userID) // wrong topic for this client

	select {
	case message := <-client.WriteChannel:
		t.Fatalf("client received a message for a topic it does not watch: %+v", message)
	case <-time.After(100 * time.Millisecond):
		// Success: nothing delivered.
	}
}

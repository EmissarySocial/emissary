package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

// Broker is a singleton. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
type Broker struct {

	// Create a map of clients, the keys of the map are the channels
	// over which we can push messages to attached clients.  (The values
	// are just booleans and are meaningless.)
	clients map[chan string]bool

	// Channel into which new clients can be pushed
	newClients chan chan string

	// Channel into which disconnected clients should be pushed
	defunctClients chan chan string

	// Channel into which messages are pushed to be broadcast out
	// to attahed clients.
	messages chan model.Stream
}

// NewBroker generates a new stream broker
func NewBroker(factory service.Factory, messages chan model.Stream) *Broker {

	result := &Broker{
		clients:        make(map[chan string]bool),
		newClients:     make(chan chan string),
		defunctClients: make(chan chan string),
		messages:       messages,
	}

	go result.Listen(factory)

	return result
}

// Listen handles
// the addition & removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
//
func (b *Broker) Listen(factory service.Factory) {

	// Get the stream service
	streamService := factory.Stream()

	// Loop endlessly
	//
	for {

		// Block until we receive from one of the
		// three following channels.
		select {

		case s := <-b.newClients:

			// There is a new client attached and we
			// want to start sending them messages.
			b.clients[s] = true
			log.Println("Added new client")

		case s := <-b.defunctClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(b.clients, s)
			close(s)

			log.Println("Removed client")

		case stream := <-b.messages:

			// TODO: need to filter only the right streams to the right receivers.
			// TODO: need to have a "view" parameter somehow
			if html, err := streamService.Render(&stream, ""); err == nil {

				// There is a new message to send.  For each
				// attached client, push the new message
				// into the client's message channel.
				for s := range b.clients {
					s <- html
				}
			}
		}
	}
}

func ServerSentEvent(b *Broker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		r := ctx.Request()
		w := ctx.Response().Writer

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)

		if !ok {
			return derp.New(500, "handler.ServerSentEvent", "Streaming Not Supported")
		}

		// Create a new channel, over which the broker can
		// send this client messages.
		messageChan := make(chan string)

		// Add this client to the map of those that should
		// receive updates
		b.newClients <- messageChan

		// Listen to the closing of the http connection via the CloseNotifier
		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			// Remove this client from the map of attached clients
			// when `EventHandler` exits.
			b.defunctClients <- messageChan
			log.Println("HTTP connection just closed.")
		}()

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		// Don't close the connection, instead loop endlessly.
		for {

			// Read from our messageChan.
			msg, open := <-messageChan

			if !open {
				// If our messageChan was closed, this means that the client has disconnected.
				break
			}

			msg = `<div id="stream">` + msg + `</div>`

			// Write to the ResponseWriter, `w`.
			fmt.Fprintf(w, "event: EventName\n")
			fmt.Fprintf(w, "data: %s\n", msg)

			// Flush the response.  This is only possible if the response supports streaming.
			f.Flush()
		}

		// Done.
		log.Println("Finished HTTP request at ", r.URL.Path)

		return nil
	}
}

func Websocket(b *Broker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		websocket.Handler(func(ws *websocket.Conn) {

			// Make a new channel to receive new messages
			messageChan := make(chan string)

			// Register the channel with the broker
			b.newClients <- messageChan

			// When complete, close the channel
			defer func() {
				b.defunctClients <- messageChan
				ws.Close()
			}()

			// Wait for new messages
			for {

				// Receive the next message from the channel
				msg, open := <-messageChan

				// If the channel has closed, then close the connection
				if !open {
					break
				}

				// Hacky wrap for websocket connection.
				msg = `<div id="stream" hx-ws="connect ws://localhost/ws">` + msg + `</div>`

				// Try to send the message to the client.
				if err := websocket.Message.Send(ws, msg); err != nil {
					return
				}
			}

		}).ServeHTTP(ctx.Response(), ctx.Request())

		return nil
	}
}

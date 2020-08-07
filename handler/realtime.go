package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ServerSentEvent(b *service.RealtimeBroker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		r := ctx.Request()
		w := ctx.Response().Writer

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)

		if !ok {
			return derp.New(500, "handler.ServerSentEvent", "Streaming Not Supported")
		}

		streamID, err := primitive.ObjectIDFromHex(ctx.Param("token"))

		if err != nil {
			return derp.Wrap(err, "ghost.handler.ServerSentEvent", "Invalid StreamID", ctx.Param("token"))
		}

		view := ctx.Param("view")

		client := service.NewRealtimeClient(streamID, view)

		// Add this client to the map of those that should
		// receive updates
		b.AddClient <- client

		// Listen to the closing of the http connection via the CloseNotifier
		if closeNotifier, ok := w.(http.CloseNotifier); ok {
			notify := closeNotifier.CloseNotify()
			go func() {
				<-notify
				// Remove this client from the map of attached clients
				// when `EventHandler` exits.
				b.RemoveClient <- client
				log.Println("HTTP connection just closed.")
			}()
		}

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		// TODO: Add a timer that fires (once a minute?) to verify that the connection is still open
		// go func(){}()

		// Don't close the connection, instead loop endlessly.
		for {

			// Read from our messageChan.
			msg, open := <-client.WriteChannel

			if !open {
				// If our messageChan was closed, this means that the client has disconnected.
				break
			}

			msg = `<div id="stream-updates">` + msg + `</div>`
			spew.Dump(msg)

			// Write to the ResponseWriter, `w`.
			fmt.Fprintf(w, "event: EventName\n")
			fmt.Fprintf(w, "data: %s\n\n", msg)

			// Flush the response.  This is only possible if the response supports streaming.
			f.Flush()
		}

		// Done
		// b.RemoveClient <- client
		log.Println("Finished HTTP request at ", r.URL.Path)

		return nil
	}
}

/*
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
*/

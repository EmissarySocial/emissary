package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// ServerSentEvent generates an echo.HandlerFunc that listens for requests for
// SSE subscriptions.
func ServerSentEvent(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return err
		}

		b := factory.RealtimeBroker()

		r := ctx.Request()
		w := ctx.Response().Writer
		done := ctx.Request().Context().Done()

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)

		if !ok {
			return derp.Report(derp.New(500, "handler.ServerSentEvent", "Streaming Not Supported"))
		}

		token := ctx.Param("stream")
		view := ctx.Param("view")

		if view == "" {
			view = "default"
		}

		httpRequest := domain.NewHTTPRequest(r)
		client := domain.NewRealtimeClient(httpRequest, token, view)

		// Add this client to the map of those that should
		// receive updates
		b.AddClient <- client

		// Guarantee that we remove this client from the broker before we leave.
		defer func() {
			b.RemoveClient <- client
		}()

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		fmt.Println("handler.realtime: connected new client to token:" + token + ", view:" + view)

		// Don't close the connection, instead loop endlessly.
		for {

			select {
			case <-done:
				log.Println("HTTP connection closed.")
				return nil

			// Read from our messageChan.
			case streamID, open := <-client.WriteChannel:

				// If our messageChan was closed, this means that the client has disconnected.
				if !open {
					return nil
				}

				// Write to the ResponseWriter, `w`.
				// eventName := "EventName1"
				// fmt.Fprintf(w, "event: %s\n", stream.StreamID)
				fmt.Fprintf(w, "data: %s\n\n", streamID.Hex())

				// Flush the response.  This is only possible if the response supports streaming.
				f.Flush()

				fmt.Println("handler.ServerSentEvents: stream sent to client: " + client.Token)
			}
		}
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

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
		w := ctx.Response().Writer
		done := ctx.Request().Context().Done()

		// Make sure that the writer supports flushing.
		f, ok := w.(http.Flusher)

		if !ok {
			return derp.Report(derp.New(500, "handler.ServerSentEvent", "Streaming Not Supported"))
		}

		token := ctx.Param("stream")

		httpRequest := domain.NewHTTPRequest(ctx)
		client := domain.NewRealtimeClient(httpRequest, token)

		// Add this client to the map of those that should
		// receive updates
		b.AddClient <- client

		// Guarantee that we remove this client from the broker before we leave.
		defer func() {
			b.RemoveClient <- client
		}()

		// Set the headers related to event streaming.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		f.Flush()

		fmt.Println("handler.realtime: connected new client to token:" + token)

		// Don't close the connection, instead loop until the client closes it (via <-done).
		for {

			select {
			case <-done:
				log.Println("HTTP connection closed.")
				return nil

			// Read from our messageChan.
			case streamID, open := <-client.WriteChannel:

				fmt.Println("handler.ServerSentEvent.  Received update for streamID: " + streamID.Hex())

				// If our messageChan was closed, this means that the client has disconnected.
				if !open {
					fmt.Println("Not Open.  Cancelling.")
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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ServerSentEvent generates an echo.HandlerFunc that listens for requests for
// SSE following.
func ServerSentEvent(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Close SSE connections that remain open after 15 minutes
	timeoutContext, cancel := context.WithTimeout(ctx.Request().Context(), 15*time.Minute)
	defer cancel()

	b := factory.RealtimeBroker()
	w := ctx.Response().Writer
	done := timeoutContext.Done()

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)

	if !ok {
		return derp.InternalError("handler.ServerSentEvent", "Streaming Not Supported")
	}

	token := ctx.Param("objectId")

	streamID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "handler.ServerSentEvent", "Invalid StreamID", token)
	}

	httpRequest := domain.NewHTTPRequest(ctx)
	client := domain.NewRealtimeClient(httpRequest, streamID)

	// Add this client to the map of those that should
	// receive updates
	b.AddClient <- client

	// Guarantee that we remove this client from the broker before we leave.
	defer func() {
		b.RemoveClient <- client
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", model.MimeTypeEventStream)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	f.Flush()

	// Don't close the connection, instead loop until the client closes it (via <-done).
	for {

		select {

		case <-done:
			return nil

		// Read from our messageChan.
		case _, open := <-client.WriteChannel:

			// If our messageChan was closed, this means that the client has disconnected.
			if !open {
				return nil
			}

			// Write to the ResponseWriter, `w`.
			fmt.Fprintf(w, "event: %s\n", streamID.Hex())
			fmt.Fprintf(w, "data: updated\n\n")

			// Flush the response.  This is only possible if the response supports streaming.
			f.Flush()
		}
	}
}

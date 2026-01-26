package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ServerSentEvent(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {
	return serverSentEvent(ctx, factory, realtime.TopicAll)
}

func ServerSentEvent_ChildUpdated(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {
	return serverSentEvent(ctx, factory, realtime.TopicChildUpdated)
}

func ServerSentEvent_FollowingUpdated(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {

	if user.UserID.Hex() != ctx.Param("objectId") {
		return derp.Forbidden("handler.ServerSentEvent_FollowingUpdated", "You do not have permission to access this resource")
	}

	return serverSentEvent(ctx, factory, realtime.TopicFollowingUpdated)
}

func ServerSentEvent_NewReplies(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {
	return serverSentEvent(ctx, factory, realtime.TopicNewReplies)
}

func ServerSentEvent_ImportProgress(ctx *steranko.Context, factory *service.Factory, _ data.Session, _ *model.User) error {
	return serverSentEvent(ctx, factory, realtime.TopicImportProgress)
}

func ServerSentEvent_Updated(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {
	return serverSentEvent(ctx, factory, realtime.TopicUpdated)
}

// ServerSentEvent generates an echo.HandlerFunc that listens for requests for
// SSE following.
func serverSentEvent(ctx *steranko.Context, factory *service.Factory, topic int) error {

	const location = "handler.ServerSentEvent"

	// Close SSE connections that remain open after 10 minutes
	timeoutContext, cancel := context.WithTimeout(ctx.Request().Context(), 10*time.Minute)
	defer cancel()

	b := factory.RealtimeBroker()
	w := ctx.Response().Writer
	done := timeoutContext.Done() // nolint:scopeguard

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)

	if !ok {
		return derp.Internal(location, "Streaming Not Supported")
	}

	token := ctx.Param("objectId")

	objectID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid StreamID", token)
	}

	client := realtime.NewClient(ctx.Request(), objectID, topic)

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
	w.Header().Set("Cache-Control", "no-store")
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
			if _, err := fmt.Fprintf(w, "event: %s\n", objectID.Hex()); err != nil {
				return derp.Wrap(err, location, "Unable to write event to response")
			}

			if _, err := fmt.Fprintf(w, "data: updated\n\n"); err != nil {
				return derp.Wrap(err, location, "Unable to write data to response")
			}

			// Flush the response.  This is only possible if the response supports streaming.
			f.Flush()
		}
	}
}

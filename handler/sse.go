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

//////////////////////////////////////////
// Object SSE Handlers
//////////////////////////////////////////

// ServerSentEvent_Object_ImportProgress streams "import progress" events for a single Import owned by the signed-in User
func ServerSentEvent_Object_ImportProgress(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.ServerSentEvent_Object_ImportProgress"

	// Parse the ImportID from the URL
	importID, err := primitive.ObjectIDFromHex(ctx.Param("objectId"))
	if err != nil {
		return derp.NotFound(location, "Invalid ObjectID", "ObjectID must be a valid ObjectID")
	}

	// RULE: The Import must belong to the signed-in User (LoadByID scopes the query to the owner)
	importRecord := model.NewImport()
	if err := factory.Import().LoadByID(session, user.UserID, importID, &importRecord); err != nil {
		return derp.Wrap(err, location, "Import not found", derp.WithNotFound())
	}

	// Stream progress events for this Import
	return serverSentEvent(ctx, factory, importRecord.ImportID, realtime.TopicImportProgress)
}

//////////////////////////////////////////
// Stream SSE Handlers
//////////////////////////////////////////

// ServerSentEvent_Stream streams every realtime topic for a Stream
func ServerSentEvent_Stream(ctx *steranko.Context, factory *service.Factory, _ data.Session, stream *model.Stream) error {
	return serverSentEvent(ctx, factory, stream.StreamID, realtime.TopicAll)
}

// ServerSentEvent_Stream_ChildUpdated streams "child updated" events for a Stream
func ServerSentEvent_Stream_ChildUpdated(ctx *steranko.Context, factory *service.Factory, _ data.Session, stream *model.Stream) error {
	return serverSentEvent(ctx, factory, stream.StreamID, realtime.TopicChildUpdated)
}

// ServerSentEvent_Stream_NewReplies streams "new replies" events for a Stream
func ServerSentEvent_Stream_NewReplies(ctx *steranko.Context, factory *service.Factory, _ data.Session, stream *model.Stream) error {
	return serverSentEvent(ctx, factory, stream.StreamID, realtime.TopicNewReplies)
}

// ServerSentEvent_Stream_Updated streams "updated" events for a Stream
func ServerSentEvent_Stream_Updated(ctx *steranko.Context, factory *service.Factory, _ data.Session, stream *model.Stream) error {
	return serverSentEvent(ctx, factory, stream.StreamID, realtime.TopicUpdated)
}

//////////////////////////////////////////
// "Me" SSE Handlers
//////////////////////////////////////////

// ServerSentEvent_Me streams every realtime topic for the signed-in User
func ServerSentEvent_Me(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicAll)
}

// ServerSentEvent_Me_FollowingUpdated streams "following updated" events for the signed-in User
func ServerSentEvent_Me_FollowingUpdated(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicFollowingUpdated)
}

// ServerSentEvent_Me_Inbox streams inbox-activity events for the signed-in User
func ServerSentEvent_Me_Inbox(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicInboxActivity)
}

// ServerSentEvent_Me_Notifications streams notification events for the signed-in User
func ServerSentEvent_Me_Notifications(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicNotification)
}

// ServerSentEvent_Me_Inbox_DirectMessage streams direct-message events for the signed-in User
func ServerSentEvent_Me_Inbox_DirectMessage(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicInboxActivity_DirectMessage)
}

// ServerSentEvent_Me_Inbox_DirectMessage_MLS streams MLS-encrypted direct-message events for the signed-in User
func ServerSentEvent_Me_Inbox_DirectMessage_MLS(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicInboxActivity_DirectMessage_MLS)
}

// ServerSentEvent_Me_Updated streams "updated" events for the signed-in User
func ServerSentEvent_Me_Updated(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return serverSentEvent(ctx, factory, user.UserID, realtime.TopicUpdated)
}

// serverSentEvent opens a Server-Sent Event stream that relays realtime messages for a single object and topic
func serverSentEvent(ctx *steranko.Context, factory *service.Factory, objectID primitive.ObjectID, topic int) error {

	const location = "handler.serverSentEvent"

	// Cap the lifetime of an SSE connection at 30 days.  This is effectively "never time
	// out" for a normal session; it exists only as a backstop so a permanently-abandoned
	// connection is eventually reclaimed.
	timeoutContext, cancel := context.WithTimeout(ctx.Request().Context(), 30*24*time.Hour)
	defer cancel()

	b := factory.RealtimeBroker()
	w := ctx.Response().Writer
	done := timeoutContext.Done() // nolint:scopeguard

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)

	if !ok {
		return derp.Internal(location, "Streaming Not Supported")
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
		case message, open := <-client.WriteChannel:

			// If our messageChan was closed, this means that the client has disconnected.
			if !open {
				return nil
			}

			// Add message ID if not empty
			if message.Event != "" {
				if _, err := fmt.Fprintf(w, "event: %s\n", message.Event); err != nil {
					return derp.Wrap(err, location, "Unable to write event to response")
				}
			}

			// Add message data
			if _, err := fmt.Fprintf(w, "data: %s\n\n", message.Data); err != nil {
				return derp.Wrap(err, location, "Unable to write data to response")
			}

			// Flush the response.  This is only possible if the response supports streaming.
			f.Flush()
		}
	}
}

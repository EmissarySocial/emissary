package mastodon

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

// streamingPingInterval is how often an idle stream is pinged, well inside proxies' idle timeouts
// (Cloudflare drops a silent websocket after about 100 seconds)
const streamingPingInterval = 30 * time.Second

// Streams that clients subscribe to by name
const (
	streamUser             = "user"
	streamUserNotification = "user:notification"
	streamDirect           = "direct"
)

// GetStreaming serves Mastodon's streaming API over a websocket. It is not in toot.
// https://docs.joinmastodon.org/methods/streaming/
func GetStreaming(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.mastodon.GetStreaming"

	return func(ctx echo.Context) error {

		request := ctx.Request()

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(serverFactory.Hostname(request))

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// RULE: Check the token while this is still an HTTP request, so a bad one gets a real 401
		request.Header.Set("Authorization", "Bearer "+streamingToken(request))
		auth, err := Authorizer(serverFactory)(request)

		if err != nil {
			return derp.Wrap(err, location, "Authorizing stream")
		}

		// Auth is already done and apps send no Origin, so the handshake accepts every request.
		// A single requested subprotocol (the token) is echoed back by the library.
		streamServer := websocket.Server{
			Handshake: func(*websocket.Config, *http.Request) error { return nil },
			Handler: func(ws *websocket.Conn) {
				serveStream(factory, auth, ws)
			},
		}

		streamServer.ServeHTTP(ctx.Response(), request)
		return nil
	}
}

// serveStream relays realtime messages for one User to one websocket until either side closes it
func serveStream(factory *service.Factory, auth model.Authorization, ws *websocket.Conn) {

	// A close error only means the socket is already gone, and there is nobody left to tell
	defer func() { _ = ws.Close() }()

	streamContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Streams may also be chosen in the URL (?stream=user)
	subs := newStreamingSubscriptions()

	if stream := ws.Request().URL.Query().Get("stream"); stream != "" {
		subs.set(stream, true)
	}

	// Attach to this Domain's realtime broker
	broker := factory.RealtimeBroker()
	client := realtime.NewClient(ws.Request(), auth.UserID, realtime.TopicAll)
	broker.AddClient <- client

	defer func() {
		broker.RemoveClient <- client
	}()

	// Read subscribe/unsubscribe commands until the client goes away
	go readStreamCommands(ws, subs, cancel)

	ticker := time.NewTicker(streamingPingInterval)
	defer ticker.Stop()

	// RULE: Only this loop writes to the websocket, so frames never interleave
	for {

		select {

		case <-streamContext.Done():
			return

		case message := <-client.WriteChannel:

			events := streamingEventsFor(factory, auth, message, subs)

			for _, event := range events {
				if err := websocket.JSON.Send(ws, event); err != nil {
					return
				}
			}

		case <-ticker.C:

			ws.PayloadType = websocket.PingFrame
			_, err := ws.Write(nil)
			ws.PayloadType = websocket.TextFrame

			if err != nil {
				return
			}
		}
	}
}

// readStreamCommands applies a client's subscribe/unsubscribe commands, and cancels the stream on any read error
func readStreamCommands(ws *websocket.Conn, subs *streamingSubscriptions, cancel context.CancelFunc) {

	defer cancel()

	for {

		var command struct {
			Type   string `json:"type"`
			Stream string `json:"stream"`
		}

		if err := websocket.JSON.Receive(ws, &command); err != nil {
			return
		}

		switch command.Type {
		case "subscribe":
			subs.set(command.Stream, true)
		case "unsubscribe":
			subs.set(command.Stream, false)
		}
	}
}

// streamingEventsFor opens a short database session and builds the events a realtime message calls for
func streamingEventsFor(factory *service.Factory, auth model.Authorization, message realtime.Message, subs *streamingSubscriptions) []streamingEvent {

	// RULE: A stream can stay open for hours, so each message gets its own session
	session, cancel, err := factory.Session(time.Minute)

	if err != nil {
		derp.Report(derp.Wrap(err, "handler.mastodon.streamingEventsFor", "Creating session"))
		return nil
	}

	defer cancel()

	return streamingEventsForMessage(factory, session, auth, message, subs)
}

// streamingEventsForMessage builds the events a realtime message calls for: none when this client
// is not subscribed to them or the record cannot be loaded
func streamingEventsForMessage(factory *service.Factory, session data.Session, auth model.Authorization, message realtime.Message, subs *streamingSubscriptions) []streamingEvent {

	if message.RecordID.IsZero() {
		return nil
	}

	switch message.Topic {

	case realtime.TopicNewsItem:

		if !subs.has(streamUser) {
			return nil
		}

		// RULE: The record is reloaded for this User only, so a nudge can never leak another User's data
		newsItem := model.NewNewsItem()

		if err := factory.NewsFeed().LoadByID(session, auth.UserID, message.RecordID, &newsItem); err != nil {
			return nil
		}

		statuses := newsItemsToToots(factory, session, auth, []model.NewsItem{newsItem})

		if len(statuses) != 1 {
			return nil
		}

		return appendStreamingEvent(nil, streamUser, "update", statuses[0])

	case realtime.TopicNotification:

		// Mastodon sends notifications on "user", and on "user:notification" for clients that want only those
		notificationStream := ""

		if subs.has(streamUser) {
			notificationStream = streamUser
		} else if subs.has(streamUserNotification) {
			notificationStream = streamUserNotification
		}

		// A received direct message also updates its conversation on "direct"
		wantsConversation := subs.has(streamDirect)

		if notificationStream == "" && !wantsConversation {
			return nil
		}

		notification := model.NewNotification()

		if err := factory.Notification().LoadByID(session, auth.UserID, message.RecordID, &notification); err != nil {
			return nil
		}

		var events []streamingEvent

		if notificationStream != "" {
			events = appendStreamingEvent(events, notificationStream, "notification", notificationToToot(factory, session, auth, notification))
		}

		if wantsConversation && notification.Type == model.NotificationTypeDirect && notification.Subtype != model.NotificationSubtypeMLS {
			events = appendConversationEvent(events, factory, session, auth, notification.Actor.ProfileURL)
		}

		return events

	case realtime.TopicDirectMessageSent:

		if !subs.has(streamDirect) {
			return nil
		}

		// RULE: LoadByID is scoped to this User, so only their own sent messages are read
		sent := model.NewObject()

		if err := factory.Object().LoadByID(session, auth.UserID, message.RecordID, &sent); err != nil {
			return nil
		}

		// One conversation per recipient
		var events []streamingEvent

		for _, recipient := range sent.Permissions {
			events = appendConversationEvent(events, factory, session, auth, recipient)
		}

		return events
	}

	return nil
}

// appendConversationEvent adds a "conversation" event for the User's conversation with one person,
// skipping it when that conversation cannot be built
func appendConversationEvent(events []streamingEvent, factory *service.Factory, session data.Session, auth model.Authorization, actorURL string) []streamingEvent {

	conversation, err := conversationForActor(factory, session, auth, actorURL)

	if err != nil {
		derp.Report(err)
		return events
	}

	return appendStreamingEvent(events, streamDirect, "conversation", conversation)
}

// appendStreamingEvent encodes an entity as an event and adds it to the list, reporting and skipping one that fails to encode
func appendStreamingEvent(events []streamingEvent, stream string, name string, entity any) []streamingEvent {

	event, err := newStreamingEvent(stream, name, entity)

	if err != nil {
		derp.Report(err)
		return events
	}

	return append(events, event)
}

// streamingToken returns the access token from the websocket subprotocol, the
// "access_token" query parameter, or the Authorization header, in that order
func streamingToken(request *http.Request) string {

	if protocol := strings.TrimSpace(request.Header.Get("Sec-WebSocket-Protocol")); protocol != "" {
		return protocol
	}

	if token := request.URL.Query().Get("access_token"); token != "" {
		return token
	}

	return strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
}

// streamingEvent is one message sent to a streaming client. Payload holds the entity as a JSON string.
type streamingEvent struct {
	Stream  []string `json:"stream"`
	Event   string   `json:"event"`
	Payload string   `json:"payload"`
}

// newStreamingEvent encodes an entity into a streamingEvent for the named stream
func newStreamingEvent(stream string, event string, entity any) (streamingEvent, error) {

	payload, err := json.Marshal(entity)

	if err != nil {
		return streamingEvent{}, derp.Wrap(err, "handler.mastodon.newStreamingEvent", "Encoding payload", event)
	}

	return streamingEvent{Stream: []string{stream}, Event: event, Payload: string(payload)}, nil
}

// streamingSubscriptions is the set of streams one client has subscribed to
type streamingSubscriptions struct {
	mutex   sync.Mutex
	streams map[string]bool
}

// newStreamingSubscriptions returns an empty set of subscriptions
func newStreamingSubscriptions() *streamingSubscriptions {
	return &streamingSubscriptions{streams: map[string]bool{}}
}

// set turns a stream on or off
func (subs *streamingSubscriptions) set(stream string, on bool) {

	subs.mutex.Lock()
	defer subs.mutex.Unlock()

	if on {
		subs.streams[stream] = true
		return
	}

	delete(subs.streams, stream)
}

// has reports whether the client is subscribed to a stream
func (subs *streamingSubscriptions) has(stream string) bool {

	subs.mutex.Lock()
	defer subs.mutex.Unlock()

	return subs.streams[stream]
}

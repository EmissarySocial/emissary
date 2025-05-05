package mastodon

import (
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/markers/

// https://docs.joinmastodon.org/methods/markers/#get
func GetMarkers(serverFactory *server.Factory) func(model.Authorization, txn.GetMarkers) (map[string]object.Marker, error) {

	const location = "handler.mastodon.GetMarkers"

	return func(auth model.Authorization, t txn.GetMarkers) (map[string]object.Marker, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get the last message in the Inbox
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadOldestUnread(auth.UserID, &message); err != nil {
			return nil, derp.Wrap(err, location, "Error loading oldest unread message")
		}

		result := map[string]object.Marker{
			"notifications": {},
			"home": {
				LastReadID: message.MessageID.Hex(),
				Version:    int(message.Revision),
				UpdatedAt:  time.Unix(message.UpdateDate, 0).UTC().Format(time.RFC3339),
			},
		}

		return result, nil
	}
}

// https://docs.joinmastodon.org/methods/markers/#create
func PostMarker(serverFactory *server.Factory) func(model.Authorization, txn.PostMarker) (map[string]object.Marker, error) {

	const location = "handler.mastodon.PostMarker"

	return func(auth model.Authorization, t txn.PostMarker) (map[string]object.Marker, error) {

		// Collect the last read date
		lastReadDate, err := strconv.ParseInt(t.Home.LastReadID, 10, 64)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid LastReadID")
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain")
		}

		// Mark messages read by date
		inboxService := factory.Inbox()
		if err := inboxService.MarkReadByDate(auth.UserID, lastReadDate); err != nil {
			return nil, derp.Wrap(err, location, "Error marking messages read")
		}

		now := time.Now().UTC().Format(time.RFC3339)

		result := map[string]object.Marker{
			"notifications": {
				LastReadID: t.Notifications.LastReadID,
				UpdatedAt:  now,
			},
			"home": {
				LastReadID: t.Home.LastReadID,
				UpdatedAt:  now,
			},
		}

		return result, nil
	}
}

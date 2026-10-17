package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Get the last message in the NewsFeed
		newsFeedService := factory.NewsFeed()
		message := model.NewNewsItem()

		if err := newsFeedService.LoadOldestUnread(session, auth.UserID, &message); err != nil {
			return nil, derp.Wrap(err, location, "Loading oldest unread message")
		}

		result := map[string]object.Marker{
			"notifications": {},
			"home": {
				LastReadID: message.NewsItemID.Hex(),
				Version:    int(message.Revision),
				UpdatedAt:  model.MastodonDate(time.UnixMilli(message.UpdateDate)), // UpdateDate is milliseconds (journal UnixMilli)
			},
		}

		return result, nil
	}
}

// https://docs.joinmastodon.org/methods/markers/#create
func PostMarker(serverFactory *server.Factory) func(model.Authorization, txn.PostMarker) (map[string]object.Marker, error) {

	const location = "handler.mastodon.PostMarker"

	return func(auth model.Authorization, t txn.PostMarker) (map[string]object.Marker, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Clients send one timeline at a time, so each is handled only when it was sent
		now := model.MastodonDate(time.Now())
		result := map[string]object.Marker{}

		if t.Home.LastReadID != "" {

			if err := markHomeRead(session, factory.NewsFeed(), auth.UserID, t.Home.LastReadID); err != nil {
				return nil, derp.Wrap(err, location, "Marking home timeline read")
			}

			result["home"] = object.Marker{LastReadID: t.Home.LastReadID, UpdatedAt: now}
		}

		// Notification read positions are not stored yet, so this marker is only echoed back
		if t.Notifications.LastReadID != "" {
			result["notifications"] = object.Marker{LastReadID: t.Notifications.LastReadID, UpdatedAt: now}
		}

		return result, nil
	}
}

// markHomeRead marks the caller's news feed read up to and including the NewsItem that lastReadID names.
func markHomeRead(session data.Session, newsFeedService *service.NewsFeed, userID primitive.ObjectID, lastReadID string) error {

	const location = "handler.mastodon.markHomeRead"

	// RULE: The marker is a NewsItem ID, as GetMarkers hands out. Anything else names nothing to mark
	newsItemID, err := primitive.ObjectIDFromHex(lastReadID)

	if err != nil {
		return nil
	}

	// RULE: The NewsItem is loaded within the caller's own feed, so a marker can never touch another User's
	newsItem := model.NewNewsItem()

	if err := newsFeedService.LoadByID(session, userID, newsItemID, &newsItem); err != nil {

		// A NewsItem that has since been removed leaves nothing to mark
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading NewsItem", lastReadID)
	}

	// Everything ranked before this item, then the item itself
	if err := newsFeedService.MarkReadByDate(session, userID, newsItem.Rank); err != nil {
		return derp.Wrap(err, location, "Marking earlier messages read")
	}

	if err := newsFeedService.MarkRead(session, &newsItem); err != nil {
		return derp.Wrap(err, location, "Marking message read", lastReadID)
	}

	// Read 'em and weep
	return nil
}

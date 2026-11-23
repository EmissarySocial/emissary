package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DeleteQuote deletes every QUOTE Notification that tells the User about one quote of one Stream.
// It leaves the User's other Notifications about the same quote post, such as a mention, in place.
func (service *Notification) DeleteQuote(session data.Session, userID primitive.ObjectID, streamID primitive.ObjectID, quoteURI string, note string) error {

	const location = "service.Notification.DeleteQuote"

	// RULE: An empty quote URI names no quote, and must not match every QUOTE Notification of the Stream
	if quoteURI == "" {
		return nil
	}

	// Find the QUOTE Notifications for this quote of this Stream
	criteria := exp.Equal("userId", userID).
		AndEqual("type", model.NotificationTypeQuote).
		AndEqual("streamId", streamID).
		AndEqual("objectUrl", quoteURI)

	notifications, err := service.Range(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Querying quote Notifications", userID, streamID, quoteURI)
	}

	// Delete each one
	for notification := range notifications {
		if err := service.Delete(session, &notification, note); err != nil {
			return derp.Wrap(err, location, "Deleting Notification", notification.NotificationID)
		}
	}

	// Unquote.
	return nil
}

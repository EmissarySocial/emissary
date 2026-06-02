package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxItem represents a single User action that is posted
// to their Outbox.  It corresponds to an ActivityPub Activity
// object. https://www.w3.org/TR/activitystreams-vocabulary/#activity-types
type OutboxItem struct {
	ActivityID primitive.ObjectID `bson:"_id"`
	ActorID    primitive.ObjectID `bson:"actorId"`    // The ID of the outbox that contains this activity (e.g. User.UserID)
	ActorType  string             `bson:"actorType"`  // The type of outbox (e.g. User, Search, etc)
	URL        string             `bson:"url"`        // The URL for this activity, if applicable
	Activity   mapof.Any          `bson:"activity"`   // The original ActivityPub activity object
	Recipients sliceof.String     `bson:"recipients"` // All IDs who should receive this activity (to, cc, bto, bcc) including indirect recipients such as Public, circles, etc.

	journal.Journal `bson:",inline"`
}

// NewOutboxItem returns a fully initialized OutboxItem
func NewOutboxItem() OutboxItem {
	return OutboxItem{
		ActivityID: primitive.NewObjectID(),
		Recipients: make([]string, 0),
		Activity:   make(map[string]any),
	}
}

// ID is a part of the data.Object interface
// It returns the string version of the ActivityID
func (item OutboxItem) ID() string {
	return item.ActivityID.Hex()
}

// CalcRecipients calculates the unique list of recipients for this OutboxItem
// by examining the `to`, `cc`, `bto`, and `bcc` properties of the original
// ActivityPub object.
func (item *OutboxItem) CalcRecipients() {

	recipients := mapof.NewBool()

	// Collect named recipients from all properties (ignore duplicates)
	for _, property := range []string{vocab.PropertyTo, vocab.PropertyCC, vocab.PropertyBTo, vocab.PropertyBCC} {
		for _, recipient := range item.Activity.GetSliceOfString(property) {
			recipients[recipient] = true
		}
	}

	// Set the value back into the Activity
	item.Recipients = recipients.Keys()
}

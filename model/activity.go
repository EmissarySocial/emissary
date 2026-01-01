package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Activity represents a single User action that is posted
// to their `outbox`.  It corresponds to an ActivityPub Activity
// object. https://www.w3.org/TR/activitystreams-vocabulary/#activity-types
type Activity struct {
	ActivityID primitive.ObjectID `bson:"_id"`
	ActorID    primitive.ObjectID `bson:"actorId"`    // The ID of the outbox that contains this activity (e.g. User.UserID)
	ActorType  string             `bson:"actorType"`  // The type of outbox (e.g. User, Search, etc)
	URL        string             `bson:"url"`        // The URL for this activity, if applicable
	Object     mapof.Any          `bson:"object"`     // The original ActivityPub activity object
	Recipients sliceof.String     `bson:"recipients"` // All IDs who should receive this activity (to, cc, bto, bcc) including indirect recipients such as as:Public, circles, etc.

	journal.Journal `bson:",inline"`
}

// NewActivity returns a fully initialized Activity
func NewActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
		Recipients: make([]string, 0),
		Object:     make(map[string]any),
	}
}

// ID is a part of the data.Object interface
// It returns the string version of the ActivityID
func (activity Activity) ID() string {
	return activity.ActivityID.Hex()
}

// CalcRecipients calculates the unique list of recipients for this Activity
// by examining the `to`, `cc`, `bto`, and `bcc` properties of the original
// ActivityPub object.
func (activity *Activity) CalcRecipients() {

	recipients := mapof.NewBool()

	// Collect named recipients from all properties (ignore duplicates)
	for _, property := range []string{vocab.PropertyTo, vocab.PropertyCC, vocab.PropertyBTo, vocab.PropertyBCC} {
		for _, recipient := range activity.Object.GetSliceOfString(property) {
			recipients[recipient] = true
		}
	}

	// Set the value back into the Activity
	activity.Recipients = recipients.Keys()
}

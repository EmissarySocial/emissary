package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxItem represents a single User action that is posted
// to their Outbox.  It corresponds to an ActivityPub Activity
// object. https://www.w3.org/TR/activitystreams-vocabulary/#activity-types
type OutboxItem struct {
	ActivityID   primitive.ObjectID `json:"activityId"   bson:"_id"`
	ActorID      primitive.ObjectID `json:"actorId"      bson:"actorId"`      // The ID of the outbox that contains this activity (e.g. User.UserID)
	ActorType    string             `json:"actorType"    bson:"actorType"`    // The type of outbox (e.g. User, Search, etc)
	URL          string             `json:"url"          bson:"url"`          // The URL for this activity, if applicable
	ActivityType string             `json:"activityType" bson:"activityType"` // The type of the activity (e.g. Create, Like, Delete)
	ObjectID     string             `json:"objectId"     bson:"objectId"`     // The URL of the activity's object, if applicable
	Activity     mapof.Any          `json:"activity"     bson:"activity"`     // The original ActivityPub activity object
	Recipients   sliceof.String     `json:"recipients"   bson:"recipients"`   // All IDs who should receive this activity (to, cc, bto, bcc) including indirect recipients such as Public, circles, etc.
	Permissions  Permissions        `json:"permissions"  bson:"permissions"`  // Groups that may read this activity in the outbox collection

	journal.Journal `json:"journal" bson:",inline"`
}

// NewOutboxItem returns a fully initialized OutboxItem
func NewOutboxItem() OutboxItem {
	return OutboxItem{
		ActivityID:  primitive.NewObjectID(),
		Recipients:  make([]string, 0),
		Activity:    make(map[string]any),
		Permissions: NewPermissions(),
	}
}

// ID is a part of the data.Object interface
// It returns the string version of the ActivityID
func (item OutboxItem) ID() string {
	return item.ActivityID.Hex()
}

/******************************************
 * JSONLDGetter Interface
 ******************************************/

// GetJSONLD returns this item's activity, without the blind recipients in `bto` and `bcc`
// It is part of the JSONLDGetter interface
func (item OutboxItem) GetJSONLD() mapof.Any {

	const location = "model.OutboxItem.GetJSONLD"

	// RULE: AS2 §5.1 forbids disclosing blind recipients. Strip a copy, because delivery
	// still reads the full addressing from the stored activity.
	result, err := hannibal.StripBlindRecipients(item.Activity)

	// RULE: An activity that cannot be stripped is not served at all. This method cannot
	// return an error, so it is reported and an empty document is returned.
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Removing blind recipients", item.ActivityID.Hex()))
		return mapof.NewAny()
	}

	// An item with no activity comes back as an empty document
	return result
}

// Created returns the creation date of this item, in Unix milliseconds
// It is part of the JSONLDGetter interface
func (item OutboxItem) Created() int64 {
	return item.CreateDate
}

/******************************************
 * Other Methods
 ******************************************/

// IsVisibleTo returns TRUE if a viewer may read this item: one who holds any of its permissions,
// the actor that published it, or an actor that it names as a recipient
func (item OutboxItem) IsVisibleTo(permissions Permissions, actorURL string) bool {

	if item.Permissions.Intersects(permissions) {
		return true
	}

	// RULE: An unsigned viewer has no identity to match against the addressing
	if actorURL == "" {
		return false
	}

	if actorURL == item.Activity.GetString(vocab.PropertyActor) {
		return true
	}

	// Only an exact match counts. A collection that the item names does not admit its members here.
	return item.Recipients.Contains(actorURL)
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

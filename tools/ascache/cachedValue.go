package ascache

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CachedValue struct {
	CachedValueID  primitive.ObjectID `bson:"_id"`
	URI            string             `bson:"uri"`                  // ID/URL of this document
	Original       mapof.Any          `bson:"original"`             // Original document, parsed as a map
	Metadata       mapof.Any          `bson:"metadata"`             // Additional metadata about this document (cache control, subscription type, etc)
	PublishedDate  int64              `bson:"published"`            // Unix epoch seconds when this document was published
	RefreshesDate  int64              `bson:"refreshes"`            // Unix epoch seconds when this document should be reloaded
	ExpiresDate    int64              `bson:"expires"`              // Unix epoch seconds when this document should be deleted
	Collection     string             `bson:"collection,omitempty"` // ID/URL of the collection that this document belongs to (user outbox, etc)
	InReplyTo      string             `bson:"inReplyTo,omitempty"`  // ID/URL of the document that this document is in reply to
	ResponseCounts mapof.Int          `bson:"responses,omitempty"`  // Map of response types to the number of each type

	journal.Journal `bson:",inline"`
}

func NewCachedValue() CachedValue {
	return CachedValue{
		CachedValueID:  primitive.NewObjectID(),
		Original:       make(mapof.Any),
		Metadata:       make(mapof.Any),
		ResponseCounts: make(mapof.Int),
	}
}

func (value CachedValue) ID() string {
	return value.CachedValueID.Hex()
}

// ShouldRefresh returns TRUE if the "RefreshesDate" is in the past.
func (value CachedValue) ShouldRefresh() bool {
	return value.RefreshesDate < time.Now().Unix()
}

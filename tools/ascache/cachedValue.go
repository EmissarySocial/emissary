package ascache

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CachedValue struct {
	CachedValueID primitive.ObjectID `bson:"_id"`
	URI           string             `bson:"uri"`
	Original      mapof.Any          `bson:"original"`
	PublishedDate int64              `bson:"published"`
	RefreshesDate int64              `bson:"refreshes"`
	ExpiresDate   int64              `bson:"expires"`
	InReplyTo     string             `bson:"inReplyTo,omitempty"`

	journal.Journal `bson:"journal,inline"`
}

func NewCachedValue() CachedValue {
	return CachedValue{
		CachedValueID: primitive.NewObjectID(),
	}
}

func (value CachedValue) ID() string {
	return value.CachedValueID.Hex()
}

// ShouldRefresh returns TRUE if the "RefreshesDate" is in the past.
func (value CachedValue) ShouldRefresh() bool {
	return value.RefreshesDate < time.Now().Unix()
}

package cache

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CachedValue struct {
	CachedValueID   primitive.ObjectID `bson:"_id"`
	URI             string             `bson:"uri"`
	JSONLD          any                `bson:"jsonld"`
	ExpirationDate  int64              `bson:"exp"`
	journal.Journal `bson:"journal,inline"`
}

func NewCachedValue() CachedValue {
	return CachedValue{
		CachedValueID: primitive.NewObjectID(),
	}
}

func (document CachedValue) ID() string {
	return document.CachedValueID.Hex()
}

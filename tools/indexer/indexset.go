package indexer

import (
	"go.mongodb.org/mongo-driver/mongo"
)

// IndexSet is a collection of MongoDB indexes, keyed by index name
type IndexSet map[string]mongo.IndexModel

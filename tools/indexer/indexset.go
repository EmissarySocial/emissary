package indexer

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type IndexSet map[string]mongo.IndexModel

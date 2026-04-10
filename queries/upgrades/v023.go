package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version23...
func Version23(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 23")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {

		if context := record.GetString("context"); context == record.GetString("url") {
			context += "/pub/context"
			record.SetString("context", context)
			return true
		}

		return false
	})
}

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
		// const location = "upgrade.Version23"
		return true
	})
}

package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version27...
func Version27(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 27")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {
		// const location = "upgrade.Version27"
		return true
	})
}

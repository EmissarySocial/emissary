package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version25...
func Version25(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 25")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {
		// const location = "upgrade.Version25"
		return true
	})
}

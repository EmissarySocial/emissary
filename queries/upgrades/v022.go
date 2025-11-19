package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version22...
func Version22(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 22")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {

		const location = "upgrade.Version22"
		return true
	})
}

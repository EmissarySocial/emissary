package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version24...
func Version24(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 24")

	return ForEachRecord(session.Collection("Domain"), func(record mapof.Any) bool {
		record["mlsMode"] = "NONE"
		return true
	})
}

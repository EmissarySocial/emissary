package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version21...
func Version21(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 21")

	return ForEachRecord(session.Collection(""), func(record mapof.Any) bool {

		return false
	})
}

package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version20...
func Version20(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 20")

	return ForEachRecord(session.Collection("Outbox"), func(record mapof.Any) error {

		return nil
	})
}

package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version17...
func Version17(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 17")
	// REMOVED IN FAVOR OF INDEX SYNC FUNCTION
	return nil
}

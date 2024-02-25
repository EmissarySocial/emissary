package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version13 updates "AttributedTo" values to be single values, not slices
func Version13(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 13")
	return nil
}

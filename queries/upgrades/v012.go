package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version12 updates "AttributedTo" values to be single values, not slices
func Version12(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 12")
	return nil
}

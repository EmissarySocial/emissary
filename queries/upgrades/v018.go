package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version18...
func Version18(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 18")
	return nil
}

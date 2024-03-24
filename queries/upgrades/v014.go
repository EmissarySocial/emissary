package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version14...
func Version14(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 14")
	return nil
}

package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version16...
func Version16(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 16")
	return nil
}

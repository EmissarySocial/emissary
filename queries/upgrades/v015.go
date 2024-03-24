package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version15...
func Version15(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 15")
	return nil
}

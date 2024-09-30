package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version19...
func Version19(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 19")
	return nil
}

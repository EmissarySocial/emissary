package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version16...
func Version16(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 16")
	_, _ = session.Collection("JWT").DeleteMany(ctx, bson.M{})
	return nil
}

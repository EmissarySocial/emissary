package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version32 gives every UserConnection with no status the PENDING status that D48 named.
//
// Before D48 a connection that was not set up carried an empty status, and NeedsSetup tested
// for that empty string. Now PENDING is a real value, assigned by the constructor and read by
// NeedsSetup, so an empty status matches NONE of the three predicates: not ready, not needing
// setup, not needing reconnect. Every row written before D48 would render as nothing at all.
func Version32(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version32"

	fmt.Println("... Version 32")

	collection := session.Collection("UserConnection")

	// An empty string and a missing key are the same state; null matches both missing and null
	filter := bson.M{"status": bson.M{"$in": []any{"", nil}}}
	update := bson.M{"$set": bson.M{"status": model.UserConnectionStatusPending}}

	result, err := collection.UpdateMany(ctx, filter, update)

	if err != nil {
		return derp.Wrap(err, location, "Updating UserConnection status")
	}

	if result == nil {
		fmt.Println("...... marked an unreported number of connections PENDING")
		return nil
	}

	fmt.Println("...... marked " + fmt.Sprint(result.ModifiedCount) + " connections PENDING")

	return nil
}

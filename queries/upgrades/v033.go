package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version33 clears legacy Stream.context values that still point at a per-stream /pub/context URL.
//
// Those legacy values were written by Version23 and are not routed. A valid context collection uses
// the user-scoped collection URL generated from a real Collection record, so the migration removes
// the stale self-reference rather than pretending it remains a valid address.
func Version33(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 33")

	collection := session.Collection("Stream")

	filter := bson.M{
		"context": bson.M{"$regex": "/pub/context$"},
	}

	update := bson.M{"$set": bson.M{"context": ""}}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	if result == nil {
		fmt.Println("...... cleared an unreported number of legacy context URLs")
		return nil
	}

	fmt.Println("...... cleared " + fmt.Sprint(result.ModifiedCount) + " legacy context URLs")
	return nil
}

// NOTE: This helper keeps the migration in the same package conventions as the other upgrades.
func legacyContextURL(record mapof.Any) string {
	return record.GetString("context")
}

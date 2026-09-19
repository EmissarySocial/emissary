package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version35 renames the Follower block state from PAUSED to BLOCKED, matching Version34 on the
// other side of the relationship
func Version35(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version35"

	fmt.Println("... Version 35")

	collection := session.Collection("Follower")

	// RULE: A bare state filter is safe here.  Unlike Following, a Follower never carries the
	// new-meaning PAUSED, so there is no legacy row to tell apart from a current one.
	filter := bson.M{"stateId": "PAUSED"}
	update := bson.M{"$set": bson.M{"stateId": "BLOCKED"}}

	// Rewrite every legacy row in one pass
	result, err := collection.UpdateMany(ctx, filter, update)

	if err != nil {
		return derp.Wrap(err, location, "Renaming Follower state PAUSED to BLOCKED")
	}

	// The driver documents a non-nil result on success, but a nil here would panic below
	if result == nil {
		fmt.Println("...... renamed an unreported number of blocked Follower records")
		return nil
	}

	fmt.Println("...... renamed " + fmt.Sprint(result.ModifiedCount) + " blocked Follower records")

	// Same block, other side of the mirror
	return nil
}

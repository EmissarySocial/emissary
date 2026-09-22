package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version34 renames the Following block status from PAUSED to BLOCKED, freeing PAUSED for a
// new meaning: a source that has failed for so long that polling has backed off.
func Version34(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version34"

	fmt.Println("... Version 34")

	collection := session.Collection("Following")

	// RULE: Match on the message as well as the status.  This upgrade runs in a goroutine
	// AFTER the domain is already serving, and the new binary may write the new-meaning
	// PAUSED to a record before this runs.  The old Block() was the only writer of PAUSED
	// and always wrote exactly this message, so the pair identifies a legacy row precisely.
	filter := bson.M{
		"status":        "PAUSED",
		"statusMessage": "Paused by a block rule",
	}

	update := bson.M{"$set": bson.M{
		"status":        "BLOCKED",
		"statusMessage": "You blocked this account",
	}}

	// Rewrite every legacy row in one pass
	result, err := collection.UpdateMany(ctx, filter, update)

	if err != nil {
		return derp.Wrap(err, location, "Renaming Following status PAUSED to BLOCKED")
	}

	// The driver documents a non-nil result on success, but a nil here would panic below
	if result == nil {
		fmt.Println("...... renamed an unreported number of blocked Following records")
		return nil
	}

	fmt.Println("...... renamed " + fmt.Sprint(result.ModifiedCount) + " blocked Following records")

	// One small step for a status, one giant leap for PAUSED
	return nil
}

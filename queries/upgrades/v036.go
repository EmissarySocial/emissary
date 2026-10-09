package upgrades

import (
	"context"
	"fmt"
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version36 restores the journal dates that Version25 stripped from the Domain record, without
// which every admin save tries to INSERT a second Domain and fails with a duplicate key.
func Version36(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version36"

	fmt.Println("... Version 36")

	// RULE: Match only a missing or zero createDate, so a healthy record is never touched
	filter := bson.M{"$or": bson.A{
		bson.M{"createDate": bson.M{"$exists": false}},
		bson.M{"createDate": int64(0)},
	}}

	now := time.Now().UnixMilli()
	update := bson.M{"$set": bson.M{"createDate": now, "updateDate": now}}

	if _, err := session.Collection("Domain").UpdateMany(ctx, filter, update); err != nil {
		return derp.Wrap(err, location, "Restoring Domain journal dates")
	}

	return nil
}

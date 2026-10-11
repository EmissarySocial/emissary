package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version37 marks every Annotation with no type as an ACTOR annotation
func Version37(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version37"

	// The actor profile was the only way to create an Annotation before the Notes button,
	// so every existing record points at an actor, and no fetch is needed to say so

	fmt.Println("... Version 37")

	// RULE: Match only a missing or empty type, so a record that has one is never touched
	filter := bson.M{"$or": bson.A{
		bson.M{"type": bson.M{"$exists": false}},
		bson.M{"type": ""},
	}}

	update := bson.M{"$set": bson.M{"type": "ACTOR"}}

	if _, err := session.Collection("Annotation").UpdateMany(ctx, filter, update); err != nil {
		return derp.Wrap(err, location, "Backfilling Annotation types")
	}

	// Everybody's an actor. Even the bots. Especially the bots.
	return nil
}

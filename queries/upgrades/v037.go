package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version37 sets every Annotation's type to ACTOR or DOCUMENT
func Version37(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version37"

	// The actor profile was the only way to create an Annotation before the Notes button, so a
	// missing type means an actor.  Dev builds also wrote raw ActivityStreams types, which map
	// onto the same two values.  Both passes are safe to run again.

	fmt.Println("... Version 37")

	collection := session.Collection("Annotation")
	known := bson.A{"ACTOR", "DOCUMENT"}

	// A missing, empty, or ActivityStreams actor type becomes ACTOR
	actorFilter := bson.M{"$or": bson.A{
		bson.M{"type": bson.M{"$exists": false}},
		bson.M{"type": bson.M{"$in": bson.A{"", "Application", "Group", "Organization", "Person", "Service"}}},
	}}

	if _, err := collection.UpdateMany(ctx, actorFilter, bson.M{"$set": bson.M{"type": "ACTOR"}}); err != nil {
		return derp.Wrap(err, location, "Marking actor Annotations")
	}

	// Every other type left over is a document
	documentFilter := bson.M{"type": bson.M{"$nin": known}}

	if _, err := collection.UpdateMany(ctx, documentFilter, bson.M{"$set": bson.M{"type": "DOCUMENT"}}); err != nil {
		return derp.Wrap(err, location, "Marking document Annotations")
	}

	// Two kinds of people in this world: actors, and everything else.
	return nil
}

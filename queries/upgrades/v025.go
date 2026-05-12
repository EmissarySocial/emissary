package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version25...
func Version25(ctx context.Context, session *mongo.Database) error {

	const location = "upgrade.Version25"

	fmt.Println("... Version 25")

	// Read all Connections from the database
	cursor, err := session.Collection("Connection").Find(ctx, bson.M{})

	if err != nil {
		return derp.Wrap(err, location, "Unable to read Connections from database")
	}

	// Copy connections into a map
	connections := mapof.NewMatchable[model.Connection]()

	for cursor.Next(ctx) {
		var connection model.Connection
		if err := cursor.Decode(&connection); err != nil {
			return err
		}
		connections[connection.ProviderID] = connection
	}

	// Load the domain
	domain := model.NewDomain()

	if err := session.Collection("Domain").FindOne(ctx, bson.M{}).Decode(&domain); err != nil {
		return derp.Wrap(err, location, "Unable to read Domain from database")
	}

	// Update the domain with the new connections data
	domain.Connections = connections

	// Write the domain back to the database
	if _, err := session.Collection("Domain").ReplaceOne(ctx, bson.M{"_id": domain.DomainID}, domain); err != nil {
		return derp.Wrap(err, location, "Unable to write Domain to database")
	}

	return nil
}

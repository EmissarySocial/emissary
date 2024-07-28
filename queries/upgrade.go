package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries/upgrades"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpgradeMongoDB(connectionString string, databaseName string, domain *model.Domain) error {

	const location = "queries.UpgradeMongoDB"

	upgradeFns := []func(context.Context, *mongo.Database) error{
		nil,
		upgrades.Version1,
		upgrades.Version2,
		upgrades.Version3,
		upgrades.Version4,
		upgrades.Version5,
		upgrades.Version6,
		upgrades.Version7,
		upgrades.Version8,
		upgrades.Version9,
		upgrades.Version10,
		upgrades.Version11,
		upgrades.Version12,
		upgrades.Version13,
		upgrades.Version14,
		upgrades.Version15,
		upgrades.Version16,
	}

	// If we're already at the target database version or higher, then skip any other work
	if domain.DatabaseVersion >= uint(len(upgradeFns)-1) {
		return nil
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
	}

	session := client.Database(databaseName)

	log.Info().Msg("UPGRADING DATABASE...")

	for index, fn := range upgradeFns {

		// Skip version 00
		if fn == nil {
			continue
		}

		// Skip if this upgrade has already been run
		if domain.DatabaseVersion >= uint(index) {
			continue
		}

		// Run the upgrade
		if err := fn(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version %d", index)
		}

		// Mark the Domain as "upgraded"
		domainCollection := session.Collection("Domain")

		filter := bson.M{"_id": primitive.NilObjectID}
		update := bson.M{"$set": bson.M{"databaseVersion": index}}

		if _, err := domainCollection.UpdateOne(ctx, filter, update); err != nil {
			return derp.Wrap(err, location, "Error updating domain record")
		}
	}

	log.Info().Msg("DONE UPGRADING DATABASE")
	return nil
}

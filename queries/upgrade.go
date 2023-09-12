package queries

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	upgrades "github.com/EmissarySocial/emissary/queries/upgrades"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpgradeMongoDB(connectionString string, databaseName string, domain *model.Domain) error {

	const currentDatabaseVersion = 11
	const location = "queries.UpgradeMongoDB"

	// If we're already at the target database version, then skip any other work
	if domain.DatabaseVersion == currentDatabaseVersion {
		return nil
	}

	ctx := context.Background()
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
	}

	if err := client.Connect(ctx); err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error connecting to mongodb Server")
	}

	session := client.Database(databaseName)

	fmt.Println("UPGRADING DATABASE...")

	if domain.DatabaseVersion < 1 {
		if err := upgrades.Version1(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 1")
		}
	}

	if domain.DatabaseVersion < 2 {
		if err := upgrades.Version2(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 2")
		}
	}

	if domain.DatabaseVersion < 3 {
		if err := upgrades.Version3(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 3")
		}
	}

	if domain.DatabaseVersion < 4 {
		if err := upgrades.Version4(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 4")
		}
	}

	if domain.DatabaseVersion < 5 {
		if err := upgrades.Version5(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 5")
		}
	}

	if domain.DatabaseVersion < 6 {
		if err := upgrades.Version6(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 6")
		}
	}

	if domain.DatabaseVersion < 7 {
		if err := upgrades.Version7(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 7")
		}
	}

	if domain.DatabaseVersion < 8 {
		if err := upgrades.Version8(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 8")
		}
	}

	if domain.DatabaseVersion < 9 {
		if err := upgrades.Version9(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 9")
		}
	}

	if domain.DatabaseVersion < 10 {
		if err := upgrades.Version10(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 10")
		}
	}

	if domain.DatabaseVersion < 11 {
		if err := upgrades.Version11(ctx, session); err != nil {
			return derp.Wrap(err, location, "Error upgrading database to version 11")
		}
	}

	// Mark the Domain as "upgraded"
	domainCollection := session.Collection("Domain")

	filter := bson.M{"_id": primitive.NilObjectID}
	update := bson.M{"$set": bson.M{"databaseVersion": currentDatabaseVersion}}

	if _, err := domainCollection.UpdateOne(ctx, filter, update); err != nil {
		return derp.Wrap(err, location, "Error updating domain record")
	}

	fmt.Println(".")
	fmt.Println("DONE.")
	return nil
}

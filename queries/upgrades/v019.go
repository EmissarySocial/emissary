package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version19...
func Version19(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 19")

	return ForEachRecord(session.Collection("Outbox"), func(record mapof.Any) error {

		record["actorId"] = record["parentId"]
		record["actorType"] = record["parentType"]
		record["activityType"] = "Create"
		record["objectId"] = record["url"]
		record["permissions"] = model.NewAnonymousPermissions()

		delete(record, "parentId")
		delete(record, "parentType")
		delete(record, "type")
		delete(record, "url")

		return nil
	})
}

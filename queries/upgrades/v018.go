package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version18...
func Version18(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version18"

	fmt.Println("... Version 18")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) error {

		result := mapof.NewObject[id.Slice]()
		permissions := record.GetMap("permissions")

		for permission := range permissions {

			// Convert the permission string to an ObjectID/
			// Soft failure.. skip this record
			permissionID, err := primitive.ObjectIDFromHex(permission)

			if err != nil {
				derp.Report(derp.Wrap(err, location, "Error converting permission", permission))
				continue
			}

			// Write this permission into the result mapof.Object[id.Slice]
			roles := permissions.GetSliceOfString(permission)
			for _, role := range roles {
				result[role] = append(result[role], permissionID)
			}
		}

		// Write the result into the original record.
		record["groups"] = result
		// delete(record, "permissions")
		return nil
	})
}

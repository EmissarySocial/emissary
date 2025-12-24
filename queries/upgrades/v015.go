package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version15...
func Version15(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 15")
	{
		err := ForEachRecord(session.Collection("Follower"), func(record mapof.Any) bool { // nolint:scopeguard (readability)

			if record.GetString("stateId") == "" {
				record["stateId"] = model.FollowerStateActive
				return true
			}

			return false
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version15", "Error updating Following collection")
		}
	}
	return nil
}

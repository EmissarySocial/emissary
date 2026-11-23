package upgrades

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version38 changes nothing, and only advances the database version.
func Version38(_ context.Context, _ *mongo.Database) error {

	// Dev branches claimed slot 38 for several migrations and released none of them, so a database
	// that recorded 38 may have run any one of them. This slot stays empty (see queries/AGENTS.md).

	// These aren't the droids you're looking for
	return nil
}

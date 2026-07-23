package upgrades

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version10 is retired: it collapsed Stream and Inbox `attributedTo` slices to single values. A
// one-time cleanup (2026-07-22) zeroed every upgrade below version 20 -- the sole database in
// service is long past it, and a fresh install is born in the current schema -- so this step is
// now a no-op that only advances the database version. The slot is preserved (never renumbered) so
// stored databaseVersion values keep their meaning; see git history for the original implementation.
func Version10(_ context.Context, _ *mongo.Database) error {
	return nil
}

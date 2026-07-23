package upgrades

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version3 is retired: it regenerated every EncryptionKey keypair (originally 512-bit RSA). A
// one-time cleanup (2026-07-22) zeroed every upgrade below version 20 -- the sole database in
// service is long past it, and a fresh install is born in the current schema -- so this step is
// now a no-op that only advances the database version. The slot is preserved (never renumbered) so
// stored databaseVersion values keep their meaning; see git history for the original
// implementation. (Version26 now performs the size upgrade idempotently.)
func Version3(_ context.Context, _ *mongo.Database) error {
	return nil
}

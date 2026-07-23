package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// OAuthUserToken synchronizes the indexes on the OAuthUserToken collection, and
// purges any pre-refresh-token grants left over from the old scheme.
func OAuthUserToken(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.OAuthUserToken"

	log.Trace().Str("database", database.Name()).Str("collection", "OAuthUserToken").Msg("COLLECTION:")

	collection := database.Collection("OAuthUserToken")

	// Purge old-scheme grants. A failure here is reported but not fatal: the indexes
	// still sync, and the purge is retried on the next startup.
	if err := purgeLegacyOAuthTokens(ctx, collection); err != nil {
		derp.Report(derp.Wrap(err, location, "Purging legacy OAuth tokens", database.Name()))
	}

	return indexer.Sync(ctx, collection, indexer.IndexSet{

		// idx_OAuthUserToken_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_OAuthUserToken_Recycle": recycleIndex(),

		// Supports LoadByID / LoadByUserAndScope (userId-scoped lookups).
		"idx_OAuthUserToken_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
			},
		},

		// Supports LoadByClientAndID and DeleteByClient (clientId-scoped lookups).
		"idx_OAuthUserToken_Client": mongo.IndexModel{
			Keys: bson.D{
				{Key: "clientId", Value: 1},
			},
		},
	})
}

// purgeLegacyOAuthTokens deletes every grant created under the old scheme, in which
// the access-token JWT was persisted in a "token" field and never expired. The new
// scheme never writes that field (the access token is a transient, stateless JWT),
// so its presence uniquely identifies an old record. Pending authorization codes and
// active new grants are untouched.
//
// This is effectively a one-time deploy migration: after the first run, no record
// carries a "token" field, so later startups match nothing. The collection is small
// (one row per active grant), so the unindexed filter scan is negligible.
func purgeLegacyOAuthTokens(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.sync.purgeLegacyOAuthTokens"

	result, err := collection.DeleteMany(ctx, bson.M{"token": bson.M{"$exists": true}})

	if err != nil {
		return derp.Wrap(err, location, "Deleting legacy OAuth tokens")
	}

	// RULE: the mongo driver returns a non-nil result whenever err is nil, but nilaway
	// can't prove that contract, so guard explicitly before reading DeletedCount.
	if result == nil {
		return nil
	}

	if result.DeletedCount > 0 {
		log.Info().
			Str("database", collection.Database().Name()).
			Int64("count", result.DeletedCount).
			Msg("Purged legacy (non-expiring) OAuth tokens; affected clients must re-authorize")
	}

	return nil
}

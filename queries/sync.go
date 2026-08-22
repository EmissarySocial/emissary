package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// SyncSharedIndexes updates the MongoDB indexes in the database that every Domain shares.
//
// RULE: It works through the connection the CALLER already holds -- it opens nothing.  The old
// signature took a connect string and dialed a fresh client on every call, without ever
// disconnecting it: each configuration reload leaked a mongo client (its topology-monitor
// goroutines and pooled sockets included), on every live node, forever.  Callers own the
// context too, and should give it a deadline: against a degraded server every index operation
// waits out server selection, and this runs under the factory's reload lock.
//
// Failures are reported per collection and do not stop the rest -- one broken collection must
// not block indexes the others need -- so there is nothing left to return.
func SyncSharedIndexes(ctx context.Context, session *mongo.Database) {

	log.Trace().Msg("** BEGIN SYNCING SHARED INDEXES")

	if err := sync.DigitalDome(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Document(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.ErrorLog(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Log(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Queue(ctx, session); err != nil {
		derp.Report(err)
	}

	log.Trace().Msg("!! Finished syncing shared indexes")
}

// SyncDomainIndexes updates the MongoDB indexes in a single Domain's database.
//
// RULE: Same contract as SyncSharedIndexes -- it works through the connection the CALLER
// already holds, opening nothing.  The old connect-string signature dialed a fresh client per
// call and never disconnected it, leaking one client (topology goroutines, heartbeat sockets)
// for every domain at boot and every domain change after.  Failures are reported per
// collection and do not stop the rest, so there is nothing left to return.
func SyncDomainIndexes(ctx context.Context, session *mongo.Database) { // NOSONAR

	log.Trace().Msg("Syncing indexes for: " + session.Name())

	if err := sync.Annotation(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Attachment(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Circle(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Connection(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Collection(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.CollectionItem(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Conversation(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Domain(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.EncryptionKey(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Folder(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Follower(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Following(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Group(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Identity(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Inbox(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.JWT(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.KeyPackage(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Notification(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.MerchantAccount(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.NewsFeed(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Outbox(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.OAuthClient(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.OAuthUserToken(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.PushSubscription(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Privilege(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Product(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Response(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Rule(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.RuleSuppression(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.SearchQuery(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.SearchResult(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.SearchTag(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Stream(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.StreamDraft(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.User(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Webhook(ctx, session); err != nil {
		derp.Report(err)
	}

	log.Debug().Msg("Finished syncing indexes for: " + session.Name())
}

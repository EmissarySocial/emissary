package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SyncSharedIndexes(connectionString string, databaseName string) error {

	const location = "queries.SyncSharedIndexes"

	// Connect to the database
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, location, "Unable to create mongodb client")
	}

	session := client.Database(databaseName)

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

	return nil
}

func SyncDomainIndexes(connectionString string, databaseName string) error {

	const location = "queries.SyncDomainIndexes"

	// Connect to the database
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, location, "Unable to create mongodb client")
	}

	session := client.Database(databaseName)

	log.Trace().Msg("Syncing indexes for: " + databaseName)

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

	if err := sync.Mention(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.MerchantAccount(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.MLSKeyPackage(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.MLSMessage(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Outbox(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Privilege(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Response(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Rule(ctx, session); err != nil {
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

	log.Debug().Msg("Finished syncing indexes for: " + databaseName)

	return nil
}

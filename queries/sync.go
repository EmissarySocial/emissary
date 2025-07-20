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

	// Connect to the database
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
	}

	session := client.Database(databaseName)

	log.Debug().Msg("** BEGIN SYNCING SHARED INDEXES")

	if err := sync.DigitalDome(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Document(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Error(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Log(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Queue(ctx, session); err != nil {
		derp.Report(err)
	}

	log.Debug().Msg("** DONE SYNCING SHARED INDEXES")

	return nil
}

func SyncDomainIndexes(connectionString string, databaseName string) error {

	// Connect to the database
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
	}

	session := client.Database(databaseName)

	log.Debug().Msg("SYNC INDEXES FOR: " + databaseName)

	if err := sync.Attachment(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Circle(ctx, session); err != nil {
		derp.Report(err)
	}

	if err := sync.Connection(ctx, session); err != nil {
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

	log.Debug().Msg("**** done syncing indexes for: " + databaseName)

	return nil
}

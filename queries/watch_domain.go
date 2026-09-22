package queries

import (
	"context"
	"errors"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WatchDomain calls publish with the stored Domain record whenever it changes, and again every
// time the change stream (re)opens, until ctx is canceled.
func WatchDomain(ctx context.Context, server data.Server, publish func(model.Domain)) {

	const location = "queries.WatchDomain"

	watcher := changeWatcher{
		collection: "Domain",

		// UpdateLookup, because the upgrade runner writes the Domain with $set, and a plain
		// update event carries no document.
		fullDocument: options.UpdateLookup,

		onOpen: func(ctx context.Context, collection *mongo.Collection) error {
			return loadDomain(ctx, collection, publish)
		},

		// RULE: Never skip a zero DomainID.  The Domain record is stored with a zero `_id`.
		onDocument: func(_ context.Context, document bson.Raw) {

			domain := model.NewDomain()

			if err := bson.Unmarshal(document, &domain); err != nil {
				derp.Report(derp.Wrap(err, location, "Decoding Domain from change event"))
				return
			}

			publish(domain)
		},
	}

	watcher.run(ctx, server)
}

// loadDomain reads the stored Domain record and publishes it, doing nothing when none exists yet
func loadDomain(ctx context.Context, collection *mongo.Collection, publish func(model.Domain)) error {

	const location = "queries.loadDomain"

	domain := model.NewDomain()

	if err := collection.FindOne(ctx, bson.M{}).Decode(&domain); err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}

		return derp.Wrap(err, location, "Loading Domain record")
	}

	publish(domain)
	return nil
}

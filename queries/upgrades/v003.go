package upgrades

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version3 updates all public keys to 512-bit RSA keys (to hopefully match Mastodon)
func Version3(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version3"
	streamCollection := session.Collection("EncryptionKey")

	fmt.Println("... Version 3")

	cursor, err := streamCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Retrieving streams iterator")
	}

	for record := mapof.NewAny(); cursor.Next(ctx); record = mapof.NewAny() {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Decoding stream record")
		}

		// 2026-07-21: This originally upgraded to 512-bit keys, but those are
		// no longer supported by Go.  Retrofitting this to 2048 so that new
		// installs don't break.

		// Create an actual encryption key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

		if err != nil {
			return derp.Wrap(err, location, "Generating RSA key")
		}

		record["privatePEM"] = sigs.EncodePrivatePEM(privateKey)
		record["publicPEM"] = sigs.EncodePublicPEM(privateKey)

		// Save record with new public key
		filter := bson.M{"_id": record["_id"]}

		if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Updating stream record")
		}

		fmt.Print(".")
	}

	return nil
}

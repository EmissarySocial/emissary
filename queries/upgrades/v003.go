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

	const location = "queries.upgrades.Version1"
	streamCollection := session.Collection("EncryptionKey")

	fmt.Println("... Version 3")

	cursor, err := streamCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving streams iterator")
	}

	record := mapof.NewAny()

	for cursor.Next(ctx) {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Error decoding stream record")
		}

		// Create an actual encryption key
		privateKey, err := rsa.GenerateKey(rand.Reader, 512)

		if err != nil {
			return derp.Wrap(err, "model.CreateEncryptionKey", "Unable to generate RSA key")
		}

		record["privatePEM"] = sigs.EncodePrivatePEM(privateKey)
		record["publicPEM"] = sigs.EncodePublicPEM(privateKey)

		// Save record with new public key
		filter := bson.M{"_id": record["_id"]}

		if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Error updating stream record")
		}

		fmt.Print(".")
		record = mapof.NewAny()
	}

	return nil
}

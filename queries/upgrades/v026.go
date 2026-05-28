package upgrades

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version26 updates all public keys to 2048-bit RSA keys (to hopefully match Mastodon)
func Version26(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version26"
	keyCollection := session.Collection("EncryptionKey")

	fmt.Println("... Version 26")

	cursor, err := keyCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving keys iterator")
	}

	for record := mapof.NewAny(); cursor.Next(ctx); record = mapof.NewAny() {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Unable to decode key record")
		}

		// Create an actual encryption key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

		if err != nil {
			return derp.Wrap(err, location, "Unable to generate RSA key")
		}

		record["privatePEM"] = sigs.EncodePrivatePEM(privateKey)
		record["publicPEM"] = sigs.EncodePublicPEM(privateKey)
		record["updateDate"] = time.Now().Unix()

		// Save record with new public key
		filter := bson.M{"_id": record["_id"]}

		if _, err := keyCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Unable to update key record")
		}

		fmt.Print(".")
	}

	return nil
}

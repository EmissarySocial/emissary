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

// targetKeyBits is the minimum RSA modulus size for an EncryptionKey. The original Version3
// minted 512-bit keys, which Go no longer supports, so any key smaller than this must be
// regenerated to a usable size.
const targetKeyBits = 2048

// Version26 regenerates every EncryptionKey whose private key is smaller than targetKeyBits
// (to hopefully match Mastodon).
//
// It is idempotent: a key that already meets the target is left byte-for-byte untouched, so a
// second pass -- e.g. after a database-version reset -- never regenerates, and never
// invalidates, an account's existing identity key.
func Version26(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version26"
	keyCollection := session.Collection("EncryptionKey")

	fmt.Println("... Version 26")

	cursor, err := keyCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Retrieving keys iterator")
	}

	for record := mapof.NewAny(); cursor.Next(ctx); record = mapof.NewAny() {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Decoding key record")
		}

		// IDEMPOTENT: Leave keys that already meet the target size exactly as they are. This is
		// what makes a re-run safe -- regenerating a live key here mints a brand new identity
		// and breaks federation for that account.
		if !keyNeedsUpgrade(record.GetString("privatePEM")) {
			continue
		}

		// Create an actual encryption key
		privateKey, err := rsa.GenerateKey(rand.Reader, targetKeyBits)

		if err != nil {
			return derp.Wrap(err, location, "Generating RSA key")
		}

		record["privatePEM"] = sigs.EncodePrivatePEM(privateKey)
		record["publicPEM"] = sigs.EncodePublicPEM(privateKey)
		record["updateDate"] = time.Now().UnixMilli()

		// Save record with new public key
		filter := bson.M{"_id": record["_id"]}

		if _, err := keyCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Updating key record")
		}

		fmt.Print(".")
	}

	return nil
}

// keyNeedsUpgrade reports whether an EncryptionKey's stored private key must be regenerated to
// reach targetKeyBits. It returns TRUE when the key is missing, undecodable, not an RSA key, or
// smaller than the target -- and FALSE only when a valid RSA key already meets the target.
// Returning FALSE tells the caller to leave the record untouched, which is what makes Version26
// idempotent. Pure (no database) so the decision is unit-testable.
//
// RULE: An undecodable key regenerates. Such a key cannot sign anything, so the account is
// already broken; minting a fresh one is a repair, not a loss.
func keyNeedsUpgrade(privatePEM string) bool {

	// A missing key has no identity to preserve -- generate one.
	if privatePEM == "" {
		return true
	}

	// An undecodable key is unusable -- regenerate it.
	privateKey, err := sigs.DecodePrivatePEM(privatePEM)

	if err != nil {
		return true
	}

	// A non-RSA key cannot be size-checked and is not what this migration produces -- regenerate it.
	rsaKey, ok := privateKey.(*rsa.PrivateKey)

	if !ok {
		return true
	}

	// Keep the key ONLY when it already meets the target size.
	return rsaKey.N.BitLen() < targetKeyBits
}

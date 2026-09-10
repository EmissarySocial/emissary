package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Version31 lowercases the stored address of every EMAIL Follower.
//
// Follower.Save now stores addresses trimmed and lowercased, and every lookup normalizes its
// argument to match. Records written before that rule keep whatever case the visitor typed, so
// a lowercase lookup -- an unsubscribe arriving from Mailchimp, a re-subscription from the web
// form -- would silently miss every one of them. This brings the existing rows onto the same rule.
//
// Two records under one parent that differ only in case become identical after this runs. They
// were already one member as far as any mail provider is concerned, so they are counted and
// reported here, not deleted: removing a Follower is not a decision a migration should make.
func Version31(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version31"

	fmt.Println("... Version 31")

	if err := normalizeFollowerAddresses(ctx, session); err != nil {
		return derp.Wrap(err, location, "Normalizing Follower email addresses")
	}

	return nil
}

// normalizeFollowerAddresses reads every EMAIL Follower, plans which need rewriting, and
// rewrites them one by one. It is idempotent: a second run plans nothing.
func normalizeFollowerAddresses(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.normalizeFollowerAddresses"

	collection := session.Collection("Follower")

	// Only EMAIL Followers carry an address, and only they store it in profileUrl as well
	cursor, err := collection.Find(ctx, bson.M{"method": model.FollowerMethodEmail}, options.Find().SetProjection(bson.M{
		"parentId":           1,
		"actor.emailAddress": 1,
		"actor.profileUrl":   1,
	}))

	if err != nil {
		return derp.Wrap(err, location, "Listing EMAIL Followers")
	}

	records := make([]followerAddressRecord, 0)

	if err := cursor.All(ctx, &records); err != nil {
		return derp.Wrap(err, location, "Reading Followers from cursor")
	}

	updates, collisions := planFollowerAddressNormalization(records)

	for _, update := range updates {

		set := bson.M{"$set": bson.M{
			"actor.emailAddress": update.EmailAddress,
			"actor.profileUrl":   update.ProfileURL,
		}}

		if _, err := collection.UpdateOne(ctx, bson.M{"_id": update.FollowerID}, set); err != nil {
			return derp.Wrap(err, location, "Updating Follower", update.FollowerID)
		}
	}

	fmt.Printf("...... normalized %d of %d EMAIL Followers", len(updates), len(records))

	// A collision is worth a line of its own, because nothing else will ever mention it
	if collisions > 0 {
		fmt.Printf(" (%d now share an address with another Follower of the same parent)", collisions)
	}

	fmt.Println()

	return nil
}

// followerAddressRecord is the slice of a Follower that the planner needs
type followerAddressRecord struct {
	FollowerID primitive.ObjectID `bson:"_id"`
	ParentID   primitive.ObjectID `bson:"parentId"`
	Actor      struct {
		EmailAddress string `bson:"emailAddress"`
		ProfileURL   string `bson:"profileUrl"`
	} `bson:"actor"`
}

// followerAddressUpdate is one rewrite the planner has decided on
type followerAddressUpdate struct {
	FollowerID   primitive.ObjectID
	EmailAddress string
	ProfileURL   string
}

// planFollowerAddressNormalization returns the records whose stored address is not yet normalized,
// plus a count of records that will then share an address with a sibling. Pure, so unit-testable.
func planFollowerAddressNormalization(records []followerAddressRecord) ([]followerAddressUpdate, int) {

	updates := make([]followerAddressUpdate, 0)
	seen := make(map[string]int, len(records))

	for _, record := range records {

		emailAddress := model.NormalizeEmailAddress(record.Actor.EmailAddress)
		profileURL := model.NormalizeEmailAddress(record.Actor.ProfileURL)

		// Count every record against its (parent, normalized address) so a collision is seen
		// whichever of the pair is read first, and whether or not either needed rewriting.
		seen[record.ParentID.Hex()+"|"+emailAddress]++

		if (emailAddress == record.Actor.EmailAddress) && (profileURL == record.Actor.ProfileURL) {
			continue
		}

		updates = append(updates, followerAddressUpdate{
			FollowerID:   record.FollowerID,
			EmailAddress: emailAddress,
			ProfileURL:   profileURL,
		})
	}

	collisions := 0

	for _, count := range seen {
		if count > 1 {
			collisions += count
		}
	}

	return updates, collisions
}

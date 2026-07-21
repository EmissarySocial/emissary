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

// Version27 backfills the derived MatchKey on every Rule and removes the duplicates that predate the
// match-key engine, so the UNIQUE {userId, matchKey} index (idx_Rule_MatchKey) can build.
//
// Rules created before the engine carry `matchKey: null`; several such rows collide on
// (userId, null) and the index refuses to build (E11000 in tools.indexer.Sync). This runs once,
// BEFORE SyncDomainIndexes builds that index -- the run-once guarantee is the upgrade framework's
// DatabaseVersion tracking, so no index-existence check is needed here.
func Version27(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version27"

	fmt.Println("... Version 27")

	if err := reconcileRules(ctx, session); err != nil {
		return derp.Wrap(err, location, "Reconciling Rules")
	}

	return nil
}

// reconcileRules backfills the derived MatchKey on every Rule and removes the duplicates that predate
// the match-key engine, so the UNIQUE {userId, matchKey} index (idx_Rule_MatchKey) can build.
//
// It is idempotent: on data that already carries the right keys it plans no deletions and no backfills.
// That is what lets a later upgrade re-run it -- databases marked version 27 by an EARLIER occupant of
// that upgrade slot never ran this reconcile, so their legacy `matchKey: null` rows still block the
// index. Version28 calls this again to clear them.
func reconcileRules(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.reconcileRules"

	collection := session.Collection("Rule")

	// Read every Rule's identity fields.
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{
		"userId":     1,
		"type":       1,
		"trigger":    1,
		"matchKey":   1,
		"createDate": 1,
	}))

	if err != nil {
		return derp.Wrap(err, location, "Listing Rules")
	}

	records := make([]ruleRecord, 0)

	if err := cursor.All(ctx, &records); err != nil {
		return derp.Wrap(err, location, "Reading Rules from cursor")
	}

	// Work out which rules to delete and which match keys to backfill.
	deletions, backfills := planRuleReconcile(records)

	// Delete the duplicate and inert rules in a single round-trip.
	if len(deletions) > 0 {
		if _, err := collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": deletions}}); err != nil {
			return derp.Wrap(err, location, "Deleting duplicate and inert Rules")
		}
	}

	// Backfill match keys on the survivors that never had one.
	if len(backfills) > 0 {

		updates := make([]mongo.WriteModel, 0, len(backfills))

		for _, backfill := range backfills {
			updates = append(updates, mongo.NewUpdateOneModel().
				SetFilter(bson.M{"_id": backfill.RuleID}).
				SetUpdate(bson.M{"$set": bson.M{"matchKey": backfill.MatchKey}}))
		}

		if _, err := collection.BulkWrite(ctx, updates); err != nil {
			return derp.Wrap(err, location, "Backfilling Rule match keys")
		}
	}

	fmt.Printf("... Rules reconciled: deleted %d duplicate/inert, backfilled %d match keys\n", len(deletions), len(backfills))

	// The needs of the index outweigh the needs of the dupes. Or the one.
	return nil
}

// ruleRecord is the slice of a Rule that Version27 needs to recompute its match key and choose
// between a User's competing rules for one key.
type ruleRecord struct {
	RuleID     primitive.ObjectID `bson:"_id"`
	UserID     primitive.ObjectID `bson:"userId"`
	Type       string             `bson:"type"`
	Trigger    string             `bson:"trigger"`
	MatchKey   string             `bson:"matchKey"`
	CreateDate int64              `bson:"createDate"`
}

// ruleBackfill records a survivor whose legacy null match key must be written to the computed value.
type ruleBackfill struct {
	RuleID   primitive.ObjectID
	MatchKey string
}

// planRuleReconcile groups the rules by the (userId, RECOMPUTED matchKey) pair the unique index
// will enforce, then reduces each group to the rows that must be deleted and the survivors whose
// match key must be backfilled.
//
// Legacy rows all read `matchKey: null`, so the stored column cannot group duplicates -- two rules
// with DIFFERENT triggers both read null. The key is therefore recomputed from Type+Trigger and the
// grouping keyed on that. Pure (no database) so the whole plan is unit-testable.
func planRuleReconcile(records []ruleRecord) ([]primitive.ObjectID, []ruleBackfill) {

	groups := make(map[string][]ruleRecord)

	for _, record := range records {
		wantKey := model.RuleMatchKey(record.Type, record.Trigger)
		groups[record.UserID.Hex()+"|"+wantKey] = append(groups[record.UserID.Hex()+"|"+wantKey], record)
	}

	deletions := make([]primitive.ObjectID, 0)
	backfills := make([]ruleBackfill, 0)

	for _, group := range groups {
		groupDeletions, groupBackfill := reconcileRuleGroup(group)
		deletions = append(deletions, groupDeletions...)

		if groupBackfill != nil {
			backfills = append(backfills, *groupBackfill)
		}
	}

	return deletions, backfills
}

// reconcileRuleGroup reduces one (userId, matchKey) group to the rows to delete and, if the surviving
// row needs its legacy null key backfilled, the value to write.
func reconcileRuleGroup(group []ruleRecord) ([]primitive.ObjectID, *ruleBackfill) {

	// Every member shares a userId and Trigger-derived key, so recomputing from the first member
	// yields the key the whole group collapses onto.
	wantKey := model.RuleMatchKey(group[0].Type, group[0].Trigger)

	// RULE: An unrecognized Type (e.g. a legacy "CONTENT" rule) or an empty Trigger yields a match
	// key that no document can ever produce, so the rule is inert. Remove every one of them: keeping
	// even a single dead rule would only occupy a slot in the unique index -- and keeping more than
	// one would re-trigger the very collision this migration exists to clear.
	if wantKey == "" {
		deletions := make([]primitive.ObjectID, 0, len(group))
		for _, record := range group {
			deletions = append(deletions, record.RuleID)
		}
		return deletions, nil
	}

	// Keep the most recently created rule; it reflects the User's latest intent when two rules (say
	// a BLOCK and a MUTE on the same actor) collapse onto one key. This mirrors
	// service.Rule.reconcileDuplicate, which adopts the existing row on a same-key save.
	survivor := newestRule(group)

	deletions := make([]primitive.ObjectID, 0, len(group)-1)
	for _, record := range group {
		if record.RuleID != survivor.RuleID {
			deletions = append(deletions, record.RuleID)
		}
	}

	// A survivor that already carries the right key (saved through the service) needs no write.
	if survivor.MatchKey == wantKey {
		return deletions, nil
	}

	// Backfill the survivor's match key -- it was never computed (legacy null row).
	return deletions, &ruleBackfill{RuleID: survivor.RuleID, MatchKey: wantKey}
}

// newestRule returns the most recently created record in a non-empty group. Ties (rules created in
// the same millisecond) keep the first one seen -- which one survives does not matter; that exactly
// one survives does.
func newestRule(group []ruleRecord) ruleRecord {

	newest := group[0]

	for _, record := range group[1:] {
		if record.CreateDate > newest.CreateDate {
			newest = record
		}
	}

	return newest
}

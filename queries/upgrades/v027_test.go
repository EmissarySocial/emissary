package upgrades

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newRuleRecord builds one Rule row for the reconcile planner, stamped with a createDate so the
// newest-wins rule has something to sort on. A blank matchKey stands in for the legacy null column.
func newRuleRecord(userID primitive.ObjectID, ruleType string, trigger string, matchKey string, createDate int64) ruleRecord {
	return ruleRecord{
		RuleID:     primitive.NewObjectID(),
		UserID:     userID,
		Type:       ruleType,
		Trigger:    trigger,
		MatchKey:   matchKey,
		CreateDate: createDate,
	}
}

/******************************************
 * planRuleReconcile
 ******************************************/

// Legacy rows with DIFFERENT triggers all read null, but they are NOT duplicates: each is backfilled
// to its own computed key and none is deleted. This is the failure the startup error reported --
// several admin rules (NilObjectID) colliding only because their keys were never computed.
func TestPlanRuleReconcile_DistinctTriggersAllSurvive(t *testing.T) {

	admin := primitive.NilObjectID

	records := []ruleRecord{
		newRuleRecord(admin, model.RuleTypeDomain, "evil.com", "", 100),
		newRuleRecord(admin, model.RuleTypeDomain, "spam.net", "", 200),
		newRuleRecord(admin, model.RuleTypeActor, "https://bad.example/@troll", "", 300),
	}

	deletions, backfills := planRuleReconcile(records)

	require.Empty(t, deletions)
	require.Len(t, backfills, 3)

	// Every survivor is backfilled to the canonical key the engine computes.
	keys := backfillKeysByID(backfills)
	require.Equal(t, "DOMAIN:evil.com", keys[records[0].RuleID])
	require.Equal(t, "DOMAIN:spam.net", keys[records[1].RuleID])
	require.Equal(t, "ACTOR:https://bad.example/@troll", keys[records[2].RuleID])
}

// Two legacy rows for one User and one trigger ARE duplicates: the newest survives and is backfilled,
// the older is deleted.
func TestPlanRuleReconcile_DuplicatesKeepNewest(t *testing.T) {

	userID := primitive.NewObjectID()

	older := newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "", 100)
	newer := newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "", 200)

	deletions, backfills := planRuleReconcile([]ruleRecord{older, newer})

	require.Equal(t, []primitive.ObjectID{older.RuleID}, deletions)
	require.Len(t, backfills, 1)
	require.Equal(t, newer.RuleID, backfills[0].RuleID)
	require.Equal(t, "DOMAIN:evil.com", backfills[0].MatchKey)
}

// Case and scheme variations of one DOMAIN normalize to the same key, so they collapse to one
// survivor -- the reconcile mirrors the engine's normalization, not the raw trigger string.
func TestPlanRuleReconcile_NormalizedDuplicatesCollapse(t *testing.T) {

	userID := primitive.NewObjectID()

	older := newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "", 100)
	newer := newRuleRecord(userID, model.RuleTypeDomain, "https://Evil.COM:443/path", "", 200)

	deletions, backfills := planRuleReconcile([]ruleRecord{older, newer})

	require.Equal(t, []primitive.ObjectID{older.RuleID}, deletions)
	require.Len(t, backfills, 1)
	require.Equal(t, newer.RuleID, backfills[0].RuleID)
}

// A rule whose Type no longer exists ("CONTENT") or whose Trigger is empty computes to no key: it can
// never match a document, so every such inert row is deleted -- and none is backfilled.
func TestPlanRuleReconcile_InertRulesDeleted(t *testing.T) {

	userID := primitive.NewObjectID()

	content := newRuleRecord(userID, "CONTENT", "some keyword", "", 100)
	emptyTrigger := newRuleRecord(userID, model.RuleTypeDomain, "", "", 200)

	deletions, backfills := planRuleReconcile([]ruleRecord{content, emptyTrigger})

	require.ElementsMatch(t, []primitive.ObjectID{content.RuleID, emptyTrigger.RuleID}, deletions)
	require.Empty(t, backfills)
}

// The same trigger owned by two different Users is not a collision: the unique index spans (userId,
// matchKey), so both survive.
func TestPlanRuleReconcile_SameKeyDifferentUsersSurvive(t *testing.T) {

	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()

	ruleA := newRuleRecord(userA, model.RuleTypeDomain, "evil.com", "", 100)
	ruleB := newRuleRecord(userB, model.RuleTypeDomain, "evil.com", "", 100)

	deletions, backfills := planRuleReconcile([]ruleRecord{ruleA, ruleB})

	require.Empty(t, deletions)
	require.Len(t, backfills, 2)
}

// A survivor that already carries the correct key (saved through the service after the engine
// landed) needs no backfill write, even while its older null-key duplicate is deleted.
func TestPlanRuleReconcile_CorrectKeyNeedsNoBackfill(t *testing.T) {

	userID := primitive.NewObjectID()

	legacy := newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "", 100)
	current := newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "DOMAIN:evil.com", 200)

	deletions, backfills := planRuleReconcile([]ruleRecord{legacy, current})

	require.Equal(t, []primitive.ObjectID{legacy.RuleID}, deletions)
	require.Empty(t, backfills)
}

// A single already-correct rule is left entirely alone: nothing to delete, nothing to backfill.
func TestPlanRuleReconcile_CleanCollectionUntouched(t *testing.T) {

	userID := primitive.NewObjectID()

	deletions, backfills := planRuleReconcile([]ruleRecord{
		newRuleRecord(userID, model.RuleTypeDomain, "evil.com", "DOMAIN:evil.com", 100),
	})

	require.Empty(t, deletions)
	require.Empty(t, backfills)
}

// An empty collection produces an empty plan without panicking.
func TestPlanRuleReconcile_Empty(t *testing.T) {

	deletions, backfills := planRuleReconcile([]ruleRecord{})

	require.Empty(t, deletions)
	require.Empty(t, backfills)
}

/******************************************
 * newestRule
 ******************************************/

// The newest record wins regardless of its position in the group.
func TestNewestRule(t *testing.T) {

	userID := primitive.NewObjectID()

	first := newRuleRecord(userID, model.RuleTypeDomain, "a.com", "", 300)
	middle := newRuleRecord(userID, model.RuleTypeDomain, "b.com", "", 100)
	last := newRuleRecord(userID, model.RuleTypeDomain, "c.com", "", 200)

	require.Equal(t, first.RuleID, newestRule([]ruleRecord{first, middle, last}).RuleID)
	require.Equal(t, first.RuleID, newestRule([]ruleRecord{middle, last, first}).RuleID)
	require.Equal(t, first.RuleID, newestRule([]ruleRecord{middle, first, last}).RuleID)
}

// A single-record group returns that record.
func TestNewestRule_Single(t *testing.T) {

	only := newRuleRecord(primitive.NewObjectID(), model.RuleTypeDomain, "a.com", "", 100)

	require.Equal(t, only.RuleID, newestRule([]ruleRecord{only}).RuleID)
}

// backfillKeysByID indexes a backfill plan by RuleID so a test can assert the key written to each
// survivor without depending on slice order (the plan iterates a map).
func backfillKeysByID(backfills []ruleBackfill) map[primitive.ObjectID]string {

	result := make(map[primitive.ObjectID]string, len(backfills))

	for _, backfill := range backfills {
		result[backfill.RuleID] = backfill.MatchKey
	}

	return result
}

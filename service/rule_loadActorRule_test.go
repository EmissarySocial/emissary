package service

import (
	"context"
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests lock the lookup that BlockActor and UnblockActor depend on. The hazard they guard is
// silent: a lookup keyed differently than Save stored simply reports Not Found, which UnblockActor
// reads as "nothing to unblock" -- so the block survives an unblock with no error anywhere. A
// hand-built data.Collection is used for the same reason as rule_disposition_test.go.

/******************************************
 * actorRuleStore -- an in-memory data.Collection that matches model.Rule records on the fields
 * LoadByMatchKey queries: userId (IN), matchKey (equal), and the notDeleted() deleteDate guard.
 ******************************************/

// actorRuleStore is an in-memory data.Collection of Rules, used by the tests in this file
type actorRuleStore struct {
	records []model.Rule
	loaded  []string // every matchKey this store was asked for, in order
	err     error    // when present, every Load fails with this error
}

// Context implements the interface, returning a background context
func (c *actorRuleStore) Context() context.Context { return context.Background() }

// Load implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	c.loaded = append(c.loaded, criteriaMatchKey(criteria))

	if c.err != nil {
		return c.err
	}

	result, ok := target.(*model.Rule)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	for _, record := range c.records {
		if matchesActorRule(criteria, record) {
			*result = record
			return nil
		}
	}

	return derp.NotFound("test", "rule not found")
}

// Count implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Save(data.Object, string) error { return derp.Internal("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) Delete(data.Object, string) error { return derp.Internal("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *actorRuleStore) HardDelete(exp.Expression) error { return derp.Internal("test", "unused") }

// matchesActorRule reports whether a Rule satisfies the userId IN / matchKey == criteria plus the
// notDeleted() guard. Any unsupported field or operator conservatively counts as "no".
func matchesActorRule(criteria exp.Expression, record model.Rule) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "userId":
			values, ok := predicate.Value.([]primitive.ObjectID)
			return ok && (predicate.Operator == exp.OperatorIn) && slices.Contains(values, record.UserID)

		case "matchKey":
			value, ok := predicate.Value.(string)
			return ok && (predicate.Operator == exp.OperatorEqual) && (value == record.MatchKey)

		case "deleteDate":
			// All test records are live; the notDeleted() guard always passes.
			return predicate.Operator == exp.OperatorEqual

		default:
			return false
		}
	})
}

// criteriaMatchKey extracts the matchKey a criteria expression is searching for, so tests can assert
// WHICH key shapes were probed and in what order. The callback returns TRUE for every predicate
// because the criteria is an AND chain: a FALSE short-circuits the walk before it reaches matchKey.
func criteriaMatchKey(criteria exp.Expression) string {

	result := ""

	criteria.Match(func(predicate exp.Predicate) bool {
		if (predicate.Field == "matchKey") && (predicate.Operator == exp.OperatorEqual) {
			if value, ok := predicate.Value.(string); ok {
				result = value
			}
		}
		return true
	})

	return result
}

// storedRule builds a persisted Rule whose MatchKey is derived from `keyTrigger` (what Save resolved)
// while Trigger keeps `trigger` (the friendly value the User typed) -- the exact split that made the
// handle-keyed lookup miss.
func storedRule(userID primitive.ObjectID, trigger string, keyTrigger string) model.Rule {
	rule := model.NewRule()
	rule.RuleID = primitive.NewObjectID()
	rule.UserID = userID
	rule.Type = model.RuleTypeActor
	rule.Action = model.RuleActionBlock
	rule.Trigger = trigger
	rule.MatchKey = model.RuleMatchKey(model.RuleTypeActor, keyTrigger)
	return rule
}

/******************************************
 * loadActorRule
 ******************************************/

// TestLoadActorRule_HandleFindsCanonicalRule is the regression proof for the reported bug: the caller
// holds the webfinger handle, but Save stored a MatchKey derived from the canonical actor URL. The
// lookup MUST resolve before keying, or the rule is invisible to UnblockActor.
func TestLoadActorRule_HandleFindsCanonicalRule(t *testing.T) {

	const handle = "@bsky.brid.gy@bsky.brid.gy"
	const canonical = "https://bsky.brid.gy/bsky.brid.gy"

	userID := primitive.NewObjectID()
	stored := storedRule(userID, handle, canonical)

	store := &actorRuleStore{records: []model.Rule{stored}}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{result: canonical}

	// Keying on the raw handle -- what the code did before -- produces a key the stored rule does not
	// carry, which is precisely why the old lookup silently missed.
	require.NotEqual(t, model.RuleMatchKey(model.RuleTypeActor, handle), stored.MatchKey)

	rule := model.NewRule()
	err := service.loadActorRule(session, userID, handle, &rule)

	require.Nil(t, err)
	require.Equal(t, stored.RuleID, rule.RuleID)
}

// TestLoadActorRule_LegacyRawKeyStillFound proves the fallback probe: rules backfilled by the v027
// migration carry a key derived from the RAW trigger, because that migration cannot resolve handles.
// Resolution succeeds here, so the canonical probe runs first and misses -- the raw probe must follow.
func TestLoadActorRule_LegacyRawKeyStillFound(t *testing.T) {

	const handle = "@bsky.brid.gy@bsky.brid.gy"
	const canonical = "https://bsky.brid.gy/bsky.brid.gy"

	userID := primitive.NewObjectID()
	legacy := storedRule(userID, handle, handle) // key derived from the raw handle

	store := &actorRuleStore{records: []model.Rule{legacy}}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{result: canonical}

	rule := model.NewRule()
	err := service.loadActorRule(session, userID, handle, &rule)

	require.Nil(t, err)
	require.Equal(t, legacy.RuleID, rule.RuleID)

	// Both shapes were probed, canonical first.
	require.Equal(t, []string{
		model.RuleMatchKey(model.RuleTypeActor, canonical),
		model.RuleMatchKey(model.RuleTypeActor, handle),
	}, store.loaded)
}

// TestLoadActorRule_UnresolvableStillFindsRawKey proves an actor whose server has gone away remains
// unblockable: resolution fails, and the raw probe alone still finds the User's own rule.
func TestLoadActorRule_UnresolvableStillFindsRawKey(t *testing.T) {

	const handle = "@ghost@nowhere.invalid"

	userID := primitive.NewObjectID()
	legacy := storedRule(userID, handle, handle)

	store := &actorRuleStore{records: []model.Rule{legacy}}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{err: derp.NotFound("test", "no such actor")}

	rule := model.NewRule()
	err := service.loadActorRule(session, userID, handle, &rule)

	require.Nil(t, err)
	require.Equal(t, legacy.RuleID, rule.RuleID)

	// The canonical probe never ran, so the raw key is the only one queried.
	require.Equal(t, []string{model.RuleMatchKey(model.RuleTypeActor, handle)}, store.loaded)
}

// TestLoadActorRule_CanonicalCallerUnchanged proves a caller that already holds the canonical URL is
// served by the first probe -- resolution is idempotent, not a second shape to worry about.
func TestLoadActorRule_CanonicalCallerUnchanged(t *testing.T) {

	const canonical = "https://bsky.brid.gy/bsky.brid.gy"

	userID := primitive.NewObjectID()
	stored := storedRule(userID, canonical, canonical)

	store := &actorRuleStore{records: []model.Rule{stored}}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{result: canonical}

	rule := model.NewRule()
	err := service.loadActorRule(session, userID, canonical, &rule)

	require.Nil(t, err)
	require.Equal(t, stored.RuleID, rule.RuleID)
	require.Equal(t, []string{model.RuleMatchKey(model.RuleTypeActor, canonical)}, store.loaded)
}

// TestLoadActorRule_NoRuleReportsNotFound proves the "nothing to unblock" path still reports NotFound
// (which UnblockActor translates to a no-op) after both probes come up empty.
func TestLoadActorRule_NoRuleReportsNotFound(t *testing.T) {

	const handle = "@alice@example.social"
	const canonical = "https://example.social/@alice"

	store := &actorRuleStore{records: []model.Rule{}}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{result: canonical}

	rule := model.NewRule()
	err := service.loadActorRule(session, primitive.NewObjectID(), handle, &rule)

	require.NotNil(t, err)
	require.True(t, derp.IsNotFound(err))
	require.Len(t, store.loaded, 2) // both shapes were tried before giving up
}

// TestLoadActorRule_DatabaseErrorNotMasked proves a real database failure surfaces instead of being
// swallowed by the fallback probe -- otherwise an outage would read as "no rule exists" and
// UnblockActor would silently no-op while the block remained in place.
func TestLoadActorRule_DatabaseErrorNotMasked(t *testing.T) {

	const canonical = "https://example.social/@alice"

	store := &actorRuleStore{err: derp.Internal("test", "database is on fire")}
	service, session := newRuleService(store)
	service.activityStreamService = &fakeActorLoader{result: canonical}

	rule := model.NewRule()
	err := service.loadActorRule(session, primitive.NewObjectID(), "@alice@example.social", &rule)

	require.NotNil(t, err)
	require.False(t, derp.IsNotFound(err))
	require.Len(t, store.loaded, 1) // stopped at the canonical probe; no fallback masked the failure
}

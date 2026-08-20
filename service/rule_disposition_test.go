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
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests lock the query->engine wiring of DispositionForKeys/Disposition against an in-memory
// rule store. The matching LOGIC (which key matches which rule) is proven in the model package; here
// we prove the service issues the right `userId IN [...] AND matchKey IN [...]` query and feeds the
// results to the engine. A hand-built data.Collection is used for the same reason as
// response_test.go -- data-mock matches on raw bson tags and can't handle the projection.

// dispositionNow is the fixed timestamp that the rule-disposition tests below are anchored to
const dispositionNow = int64(1_000_000)

/******************************************
 * ruleStore -- an in-memory data.Collection that matches RuleSummaries on the fields
 * QueryByMatchKeys uses: userId (IN), matchKey (IN), and the notDeleted() deleteDate guard.
 ******************************************/

// ruleStore is an in-memory data.Collection of RuleSummaries, used by the tests in this file
type ruleStore struct {
	records []model.RuleSummary
}

// Context implements the interface, returning a background context
func (c *ruleStore) Context() context.Context { return context.Background() }

// Query implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Query(target any, criteria exp.Expression, _ ...option.Option) error {

	result, ok := target.(*[]model.RuleSummary)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	for _, record := range c.records {
		if matchesRule(criteria, record) {
			*result = append(*result, record)
		}
	}

	return nil
}

// Count implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.NotFound("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Save(data.Object, string) error { return derp.NotFound("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) Delete(data.Object, string) error { return derp.NotFound("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *ruleStore) HardDelete(exp.Expression) error { return derp.NotFound("test", "unused") }

// matchesRule reports whether a RuleSummary satisfies the IN criteria on userId/matchKey plus the
// notDeleted() deleteDate==0 guard. Any unsupported field or operator conservatively counts as "no".
func matchesRule(criteria exp.Expression, record model.RuleSummary) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "userId":
			values, ok := predicate.Value.([]primitive.ObjectID)
			return ok && (predicate.Operator == exp.OperatorIn) && slices.Contains(values, record.UserID)

		case "matchKey":
			values, ok := predicate.Value.([]string)
			return ok && (predicate.Operator == exp.OperatorIn) && slices.Contains(values, record.MatchKey)

		case "deleteDate":
			// All test records are live; the notDeleted() guard always passes.
			return predicate.Operator == exp.OperatorEqual

		default:
			return false
		}
	})
}

// ruleSession is a data.Session that hands out a single ruleStore
type ruleSession struct {
	store data.Collection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s ruleSession) Collection(string) data.Collection { return s.store }

// Context implements the interface, returning a background context
func (s ruleSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s ruleSession) Close() {}

// newRuleService returns a Rule service backed by the provided store
func newRuleService(store data.Collection) (*Rule, ruleSession) {
	service := NewRule()
	return &service, ruleSession{store: store}
}

// summaryRule builds a persisted RuleSummary with its MatchKey derived exactly as Save would.
func summaryRule(userID primitive.ObjectID, ruleType string, action string, trigger string) model.RuleSummary {
	return model.RuleSummary{
		RuleID:   primitive.NewObjectID(),
		UserID:   userID,
		Type:     ruleType,
		Action:   action,
		Trigger:  trigger,
		MatchKey: model.RuleMatchKey(ruleType, trigger),
	}
}

/******************************************
 * DispositionForKeys
 ******************************************/

// A user's own ACTOR block is found by the actor's keys and reported as a USER-tier block.
func TestRule_DispositionForKeys_UserBlock(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer")
	store := &ruleStore{records: []model.RuleSummary{block}}

	service, session := newRuleService(store)
	keys := model.ActorMatchKeys("https://evil.example/@spammer")

	disposition, err := service.DispositionForKeys(session, userID, keys, dispositionNow)

	require.Nil(t, err)
	require.True(t, disposition.IsBlocked())
	require.Equal(t, block.RuleID, disposition.RuleID)
	require.Equal(t, model.RuleOriginUser, disposition.Tier)
}

// A domain-wide ADMIN block (UserID zero) applies to a User's query and is reported as ADMIN-tier.
// This is the "ADMIN is a floor" theorem in action -- the same query returns both tiers.
func TestRule_DispositionForKeys_AdminDomainBlock(t *testing.T) {

	userID := primitive.NewObjectID()
	adminBlock := summaryRule(primitive.NilObjectID, model.RuleTypeDomain, model.RuleActionBlock, "evil.example")
	store := &ruleStore{records: []model.RuleSummary{adminBlock}}

	service, session := newRuleService(store)
	// A subdomain of the blocked domain -- the suffix enumeration must reach the rule.
	keys := model.ActorMatchKeys("https://mail.evil.example/@spammer")

	disposition, err := service.DispositionForKeys(session, userID, keys, dispositionNow)

	require.Nil(t, err)
	require.True(t, disposition.IsBlocked())
	require.Equal(t, model.RuleOriginAdmin, disposition.Tier)
}

// MUTE is filtered but not blocked -- the wire gate (which checks IsBlocked) lets it through, while
// newsfeed ingest (which checks IsFiltered) will not.
func TestRule_DispositionForKeys_Mute(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeActor, model.RuleActionMute, "https://noisy.example/@chatty")
	store := &ruleStore{records: []model.RuleSummary{mute}}

	service, session := newRuleService(store)
	keys := model.ActorMatchKeys("https://noisy.example/@chatty")

	disposition, err := service.DispositionForKeys(session, userID, keys, dispositionNow)

	require.Nil(t, err)
	require.False(t, disposition.IsBlocked())
	require.True(t, disposition.IsMuted())
	require.True(t, disposition.IsFiltered())
}

// An unrelated actor matches no rule -- the store has a block, but not for this origin.
func TestRule_DispositionForKeys_NoMatch(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer")
	store := &ruleStore{records: []model.RuleSummary{block}}

	service, session := newRuleService(store)
	keys := model.ActorMatchKeys("https://good.example/@friend")

	disposition, err := service.DispositionForKeys(session, userID, keys, dispositionNow)

	require.Nil(t, err)
	require.False(t, disposition.IsFiltered())
}

// Another User's rule never leaks into this User's disposition, even for the same actor.
func TestRule_DispositionForKeys_ForeignUserRuleIgnored(t *testing.T) {

	me := primitive.NewObjectID()
	someoneElse := primitive.NewObjectID()
	theirBlock := summaryRule(someoneElse, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer")
	store := &ruleStore{records: []model.RuleSummary{theirBlock}}

	service, session := newRuleService(store)
	keys := model.ActorMatchKeys("https://evil.example/@spammer")

	disposition, err := service.DispositionForKeys(session, me, keys, dispositionNow)

	require.Nil(t, err)
	require.False(t, disposition.IsFiltered())
}

/******************************************
 * IsActorBlocked
 ******************************************/

// IsActorBlocked is TRUE for a blocked actor and FALSE for a merely-muted one (MUTE never gates the
// wire) -- the shared check behind the Stage-2 gate and the Follow handlers.
func TestRule_IsActorBlocked(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer")
	mute := summaryRule(userID, model.RuleTypeActor, model.RuleActionMute, "https://noisy.example/@chatty")
	store := &ruleStore{records: []model.RuleSummary{block, mute}}

	service, session := newRuleService(store)

	blockedDoc := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://evil.example/@spammer"})
	blocked, err := service.IsActorBlocked(session, userID, blockedDoc)
	require.Nil(t, err)
	require.True(t, blocked)

	mutedDoc := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://noisy.example/@chatty"})
	muted, err := service.IsActorBlocked(session, userID, mutedDoc)
	require.Nil(t, err)
	require.False(t, muted)
}

/******************************************
 * Disposition (document convenience)
 ******************************************/

// The document path also matches a TAG rule against the document's hashtags -- proving Disposition
// feeds DocumentMatchKeys (which includes tags), unlike the actor-only wire gate.
func TestRule_Disposition_TagOnDocument(t *testing.T) {

	userID := primitive.NewObjectID()
	tagMute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#uspol")
	store := &ruleStore{records: []model.RuleSummary{tagMute}}

	service, session := newRuleService(store)

	document := streams.NewDocument(mapof.Any{
		vocab.PropertyActor: "https://good.example/@friend",
		vocab.PropertyTag: []mapof.Any{{
			vocab.PropertyType: vocab.LinkTypeHashtag,
			vocab.PropertyName: "#uspol",
		}},
	})

	disposition, err := service.Disposition(session, userID, document, dispositionNow)

	require.Nil(t, err)
	require.True(t, disposition.IsMuted())
}

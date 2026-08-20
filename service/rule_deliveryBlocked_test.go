package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// failingRuleStore errors on every query, to exercise the skip-silently posture (P5-2).
type failingRuleStore struct {
	ruleStore
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *failingRuleStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "database is on fire")
}

// TestRule_DeliveryBlocked pins the R4 egress gate: BLOCK halts delivery, MUTE never does (D5),
// admin DOMAIN blocks cover whole hosts, and unidentifiable recipients are never delivered to.
func TestRule_DeliveryBlocked(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer"),
		summaryRule(userID, model.RuleTypeActor, model.RuleActionMute, "https://noisy.example/@chatty"),
		summaryRule(primitive.NilObjectID, model.RuleTypeDomain, model.RuleActionBlock, "banned.example"),
	}}

	service, session := newRuleService(store)

	// A blocked actor is never delivered to
	require.True(t, service.DeliveryBlocked(session, userID, "https://evil.example/@spammer"))

	// RULE: MUTE never gates egress (D5)
	require.False(t, service.DeliveryBlocked(session, userID, "https://noisy.example/@chatty"))

	// An admin DOMAIN block halts delivery to anyone on that host, for every sender
	require.True(t, service.DeliveryBlocked(session, userID, "https://banned.example/@anyone"))

	// Clean recipients are delivered to
	require.False(t, service.DeliveryBlocked(session, userID, "https://friendly.example/@pal"))

	// An unidentifiable recipient cannot be cleared, so it is not delivered to
	require.True(t, service.DeliveryBlocked(session, userID, ""))
}

// TestRule_DeliveryBlocked_SkipSilently pins the P5-2 posture: a rules-query failure treats the
// recipient as blocked -- no error escapes, nothing alerts, nothing retries.
func TestRule_DeliveryBlocked_SkipSilently(t *testing.T) {

	service, _ := newRuleService(nil)
	session := ruleSession{store: &failingRuleStore{}}

	require.True(t, service.DeliveryBlocked(session, primitive.NewObjectID(), "https://any.example/@actor"))
}

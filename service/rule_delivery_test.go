package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// blockRule returns a BLOCK rule summary of the given type and trigger, with its match key calculated
func blockRule(t *testing.T, ruleType string, trigger string) model.RuleSummary {

	t.Helper()

	rule := model.NewRule()
	rule.Type = ruleType
	rule.Trigger = trigger
	rule.Action = model.RuleActionBlock
	rule.MatchKey = model.RuleMatchKey(ruleType, trigger)
	require.NotEmpty(t, rule.MatchKey)

	return model.RuleSummary{RuleID: rule.RuleID, Type: rule.Type, Action: rule.Action, Trigger: rule.Trigger, MatchKey: rule.MatchKey}
}

// TestNewDeliveryFilter verifies that blocked actors and domains are refused, by ActivityPub and by email
func TestNewDeliveryFilter(t *testing.T) {

	allowed := newDeliveryFilter([]model.RuleSummary{
		blockRule(t, model.RuleTypeActor, "https://remote.example/@blocked"),
		blockRule(t, model.RuleTypeDomain, "bad.example"),
	}, 1_800_000_000)

	// A blocked actor, in any host case, and nobody else on that host
	require.False(t, allowed("https://remote.example/@blocked"))
	require.False(t, allowed("https://REMOTE.example/@blocked"))
	require.True(t, allowed("https://remote.example/@friend"))

	// A blocked domain, its subdomains, and email to it
	require.False(t, allowed("https://bad.example/@anyone"))
	require.False(t, allowed("https://social.bad.example/@anyone"))
	require.False(t, allowed("mailto:someone@bad.example"))
	require.False(t, allowed("mailto:someone@mail.bad.example"))
	require.True(t, allowed("https://notbad.example/@anyone"))
	require.True(t, allowed("mailto:someone@good.example"))

	// A recipient that cannot be identified is not delivered to
	require.False(t, allowed(""))
	require.False(t, allowed("mailto:no-domain"))
	require.False(t, allowed("mailto:trailing@"))
}

// TestNewDeliveryFilter_NoRules verifies that every identifiable recipient passes when nothing is blocked
func TestNewDeliveryFilter_NoRules(t *testing.T) {

	allowed := newDeliveryFilter(nil, 1_800_000_000)

	require.True(t, allowed("https://remote.example/@friend"))
	require.True(t, allowed("mailto:someone@good.example"))
	require.False(t, allowed(""))
}

// TestNewDeliveryFilter_IgnoresOtherRules verifies that only a rule whose key the recipient produces counts
func TestNewDeliveryFilter_IgnoresOtherRules(t *testing.T) {

	mute := blockRule(t, model.RuleTypeActor, "https://remote.example/@muted")
	mute.Action = model.RuleActionMute

	allowed := newDeliveryFilter([]model.RuleSummary{
		mute,
		blockRule(t, model.RuleTypeActor, "https://remote.example/@someone-else"),
	}, 1_800_000_000)

	require.True(t, allowed("https://remote.example/@muted"), "a mute never gates delivery")
	require.True(t, allowed("https://remote.example/@friend"))
}

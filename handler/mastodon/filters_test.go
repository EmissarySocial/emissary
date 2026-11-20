package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestFilterFromRule confirms how hashtag Rules appear as Mastodon Filters
func TestFilterFromRule(t *testing.T) {

	muted := model.Rule{Trigger: "politics", Action: model.RuleActionMute}
	require.Equal(t, "hide", filterFromRule(muted).FilterAction)

	labeled := model.Rule{Trigger: "spoilers", Action: model.RuleActionLabel}
	require.Equal(t, "warn", filterFromRule(labeled).FilterAction)
}

// TestFilterFromRule_Title confirms a hashtag shows with exactly one "#", however it was saved
func TestFilterFromRule_Title(t *testing.T) {
	require.Equal(t, "#spoilers", filterFromRule(model.Rule{Trigger: "spoilers"}).Title)
	require.Equal(t, "#spoilers", filterFromRule(model.Rule{Trigger: "#spoilers"}).Title)
}

// TestFilterFromRule_Details confirms a hashtag Filter applies in every context, with the hashtag as its keyword
func TestFilterFromRule_Details(t *testing.T) {

	filter := filterFromRule(model.Rule{Trigger: "spoilers"})

	require.ElementsMatch(t, []string{"home", "notifications", "public", "thread", "account"}, filter.Context)
	require.Nil(t, filter.ExpiresAt)
	require.Len(t, filter.Keywords, 1)
	require.Equal(t, "#spoilers", filter.Keywords[0].Keyword)
	require.True(t, filter.Keywords[0].WholeWord)
}

// TestApplyFilter_Create confirms a new filter's title becomes the Rule's hashtag, action, label and expiry
func TestApplyFilter_Create(t *testing.T) {

	rule := model.Rule{Type: model.RuleTypeTag}
	require.NoError(t, applyFilter(&rule, "politics", nil, "warn", 3600, 1000))

	require.Equal(t, "#politics", rule.Trigger)
	require.Equal(t, model.RuleActionLabel, rule.Action)
	require.Equal(t, "#politics", rule.Label, "a LABEL Rule needs label text")
	require.Equal(t, int64(4600), rule.ExpireDate)
	require.Equal(t, "hide", filterFromRule(model.Rule{Trigger: "x", Action: model.RuleActionMute}).FilterAction)
}

// TestApplyFilter_Update confirms an update keeps what it doesn't change, and a keyword names the hashtag
func TestApplyFilter_Update(t *testing.T) {

	rule := model.Rule{Type: model.RuleTypeTag, Trigger: "#testlabel", Action: model.RuleActionLabel, Label: "my label", ExpireDate: 99}

	require.NoError(t, applyFilter(&rule, "", nil, "hide", 0, 1000))
	require.Equal(t, "#testlabel", rule.Trigger)
	require.Equal(t, model.RuleActionMute, rule.Action)
	require.Equal(t, "my label", rule.Label)
	require.Zero(t, rule.ExpireDate, "no expires_in means never")

	require.NoError(t, applyFilter(&rule, "Some title", []string{"#cats"}, "", 0, 1000))
	require.Equal(t, "#cats", rule.Trigger)
	require.Equal(t, model.RuleActionMute, rule.Action)
}

// TestApplyFilter_Rejects confirms text that isn't one hashtag, and unknown actions, are refused
func TestApplyFilter_Rejects(t *testing.T) {
	require.Error(t, applyFilter(&model.Rule{}, "two words", nil, "hide", 0, 0))
	require.Error(t, applyFilter(&model.Rule{}, "", nil, "hide", 0, 0))
	require.Error(t, applyFilter(&model.Rule{}, "cats", nil, "explode", 0, 0))
}

// TestFilterFromRule_Expiry confirms an expiring Rule reports when it expires
func TestFilterFromRule_Expiry(t *testing.T) {
	filter := filterFromRule(model.Rule{Trigger: "x", ExpireDate: 86400})
	require.NotNil(t, filter.ExpiresAt)
	require.Equal(t, "1970-01-02T00:00:00Z", *filter.ExpiresAt)
}

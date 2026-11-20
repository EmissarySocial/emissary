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

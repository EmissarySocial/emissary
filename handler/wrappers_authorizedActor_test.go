package handler

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestAuthorizedActorRefusal pins the authorized-fetch disposition gate (R10)
func TestAuthorizedActorRefusal(t *testing.T) {

	// This drives the gate function directly, so it proves the helper correct and nothing more:
	// WithAuthorizedActorAndUser is wired to no route, and the anonymous -> 401 check is inlined
	// there rather than here. Neither is covered by a passing run. (BUG-21)

	// Blocked requesters get a 404, identical to a User that does not exist
	err := authorizedActorRefusal("test", model.RuleDisposition{Action: model.RuleActionBlock, Tier: model.RuleOriginUser})
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	// RULE: MUTE never gates authorized fetch -- remote observability (D5)
	require.NoError(t, authorizedActorRefusal("test", model.RuleDisposition{Action: model.RuleActionMute, Tier: model.RuleOriginUser}))

	// Clean requesters pass
	require.NoError(t, authorizedActorRefusal("test", model.RuleDisposition{}))

	// LABEL matches alone never gate
	labeled := model.RuleDisposition{Labels: []model.RuleLabelMatch{{Label: "Politics"}}}
	require.NoError(t, authorizedActorRefusal("test", labeled))
}

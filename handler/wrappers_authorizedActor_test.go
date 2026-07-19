package handler

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestAuthorizedActorRefusal pins the authorized-fetch disposition gate (R10): blocked → 404
// (indistinguishable from a missing User), muted → served (a muted actor must never detect the
// mute), clean → served. The anonymous → 401 check is inlined in WithAuthorizedActorAndUser.
func TestAuthorizedActorRefusal(t *testing.T) {

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

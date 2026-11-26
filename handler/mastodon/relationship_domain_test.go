package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestIsDomainBlocked confirms only the User's own unexpired DOMAIN blocks count
func TestIsDomainBlocked(t *testing.T) {

	user := primitive.NewObjectID()
	block := model.RuleSummary{UserID: user, Type: model.RuleTypeDomain, Action: model.RuleActionBlock}

	require.True(t, isDomainBlocked([]model.RuleSummary{block}, user, 1000))
	require.False(t, isDomainBlocked(nil, user, 1000))

	// A server-wide block belongs to the admin, not the User
	serverWide := block
	serverWide.UserID = primitive.NilObjectID
	require.False(t, isDomainBlocked([]model.RuleSummary{serverWide}, user, 1000))

	// A mute is not a block
	mute := block
	mute.Action = model.RuleActionMute
	require.False(t, isDomainBlocked([]model.RuleSummary{mute}, user, 1000))

	// An expired block is inert, and an unexpired one still counts
	expiring := block
	expiring.ExpireDate = 500
	require.False(t, isDomainBlocked([]model.RuleSummary{expiring}, user, 1000))
	expiring.ExpireDate = 2000
	require.True(t, isDomainBlocked([]model.RuleSummary{expiring}, user, 1000))
}

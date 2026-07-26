package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestLookupProvider_NotificationChannels closes the loop on a silent data-loss path.  The
// notification settings form renders its options from this lookup group and saves them through the
// User schema, which validates against an enum in model/user_accessors.go.  An option offered here
// but missing from that enum lets the user tick the box and watch the save quietly drop it -- no
// error, no clue.  This test asserts every offered option round-trips.
//
// The "notification-channels" case reads no LookupProvider state, so a zero value is safe here.
func TestLookupProvider_NotificationChannels(t *testing.T) {

	group := LookupProvider{}.Group("notification-channels")
	require.NotNil(t, group)

	codes := group.Get()
	require.NotEmpty(t, codes)

	userSchema := schema.New(model.UserSchema())

	for _, code := range codes {
		t.Run(code.Value, func(t *testing.T) {

			require.NotEmpty(t, code.Label, "every channel needs a label for the settings form")
			require.NotEmpty(t, code.Description, "every channel needs a description for the settings form")

			user := model.NewUser()
			user.NotificationChannels = nil

			require.NoError(t, userSchema.Set(&user, "notificationChannels.0", code.Value),
				"channel %q is offered in settings but rejected by the User schema enum", code.Value)
		})
	}
}

// TestLookupProvider_NotificationChannels_Complete fails when a NotificationChannel constant exists
// that the settings form never offers -- a channel a user can never switch on.
func TestLookupProvider_NotificationChannels_Complete(t *testing.T) {

	expected := []string{
		model.NotificationChannelDirectMessage,
		model.NotificationChannelReply,
		model.NotificationChannelMentionFollowing,
		model.NotificationChannelMentionNotFollowing,
		model.NotificationChannelFollow,
		model.NotificationChannelReaction,
	}

	// The composite literal needs parens inside a `range` clause, where it would otherwise be
	// parsed as the start of the loop body.
	offered := make([]string, 0, len(expected))
	for _, code := range (LookupProvider{}).Group("notification-channels").Get() {
		offered = append(offered, code.Value)
	}

	require.ElementsMatch(t, expected, offered)
}

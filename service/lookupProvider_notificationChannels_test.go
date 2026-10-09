package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestLookupProvider_NotificationChannels asserts that every option this group offers
// round-trips through the User schema
func TestLookupProvider_NotificationChannels(t *testing.T) {

	// The settings form renders from this group and saves through the schema, whose enum
	// is model.AllNotificationChannels(). An option missing there lets the user tick the
	// box and watch the save quietly drop it -- no error, no clue.
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

	expected := model.AllNotificationChannels()

	// The composite literal needs parens inside a `range` clause, where it would
	// otherwise be parsed as the start of the loop body.
	offered := make([]string, 0, len(expected))
	for _, code := range (LookupProvider{}).Group("notification-channels").Get() {
		offered = append(offered, code.Value)
	}

	require.ElementsMatch(t, expected, offered)
}

package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/stretchr/testify/require"
)

// TestNewOwnerFromConfig verifies how the bootstrap owner account is populated from the
// domain configuration, including the default fallbacks that keep User.Save from rejecting
// a blank email and the whitespace-trimming that fixes the demo config's "admin " username.
func TestNewOwnerFromConfig(t *testing.T) {

	t.Run("FullyConfigured", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "Ben Pate",
			Username:     "benpate",
			EmailAddress: "ben@pate.org",
		}, "example.com")

		require.Equal(t, "Ben Pate", owner.DisplayName)
		require.Equal(t, "benpate", owner.Username)
		require.Equal(t, "ben@pate.org", owner.EmailAddress)
		require.True(t, owner.IsOwner)
		require.True(t, owner.IsPublic)
	})

	t.Run("BlankFieldsUseDefaults", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{}, "localhost")

		require.Equal(t, "Demo", owner.DisplayName)
		require.Equal(t, "demo", owner.Username)
		require.Equal(t, "demo@localhost", owner.EmailAddress)
		require.True(t, owner.IsOwner)
		require.True(t, owner.IsPublic)
	})

	t.Run("BlankEmailFallsBackToHostname", func(t *testing.T) {
		// A blank email on a public host still yields a valid, non-empty address so that
		// User.Save (which requires an email) succeeds.
		owner := newOwnerFromConfig(config.Owner{
			DisplayName: "Site Admin",
			Username:    "siteadmin",
		}, "example.com")

		require.Equal(t, "demo@example.com", owner.EmailAddress)
	})

	t.Run("WhitespaceIsTrimmed", func(t *testing.T) {
		// The shipped demo config used "admin " (trailing space); it must be trimmed so
		// the username is usable for sign-in and passes username validation.
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "  Demo Admin  ",
			Username:     "admin ",
			EmailAddress: "  demo@example.com ",
		}, "example.com")

		require.Equal(t, "Demo Admin", owner.DisplayName)
		require.Equal(t, "admin", owner.Username)
		require.Equal(t, "demo@example.com", owner.EmailAddress)
	})

	t.Run("WhitespaceOnlyFieldsUseDefaults", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "   ",
			Username:     "   ",
			EmailAddress: "   ",
		}, "localhost")

		require.Equal(t, "Demo", owner.DisplayName)
		require.Equal(t, "demo", owner.Username)
		require.Equal(t, "demo@localhost", owner.EmailAddress)
	})
}

// TestCalcOwnerInviteMethod verifies the policy that decides how a newly-bootstrapped
// owner receives their first password.  A known default password is only acceptable on
// localhost; public hosts must never get one.
func TestCalcOwnerInviteMethod(t *testing.T) {

	testCases := []struct {
		name        string
		isLocalhost bool
		ownerEmail  string
		expected    ownerInviteMethod
	}{
		{"LocalhostWithEmail", true, "ben@pate.org", ownerInviteLocalhost},
		{"LocalhostWithoutEmail", true, "", ownerInviteLocalhost},
		{"PublicWithEmail", false, "ben@pate.org", ownerInviteEmail},
		{"PublicWithoutEmail", false, "", ownerInviteManual},
		{"PublicWithWhitespaceEmail", false, "   ", ownerInviteManual},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := calcOwnerInviteMethod(testCase.isLocalhost, testCase.ownerEmail)
			require.Equal(t, testCase.expected, result)
		})
	}
}

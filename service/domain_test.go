package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
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

// TestNeedsHostnameStamp verifies the rule that decides when the stored Domain record's hostname
// is rewritten from the server configuration.  The configuration is authoritative, EXCEPT that a
// blank configured hostname never clears a stored one -- the setup console builds factories before
// its configuration is complete, and a cleared hostname breaks every URL the domain derives.
func TestNeedsHostnameStamp(t *testing.T) {

	testCases := []struct {
		name       string
		stored     string
		configured string
		expected   bool
	}{
		{"LegacyRecordWithNoHostname", "", "example.com", true},
		{"RenamedInSetupTool", "old.example.com", "new.example.com", true},
		{"AlreadyMatches", "example.com", "example.com", false},
		{"BlankConfigNeverClearsStored", "example.com", "", false},
		{"BothBlank", "", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := needsHostnameStamp(testCase.stored, testCase.configured)
			require.Equal(t, testCase.expected, result)
		})
	}
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

// TestNewOAuthClient_EmptyProviderID pins the guard on LoadOrCreateByProvider's error.  That
// error was discarded, and its failure paths return a ZERO Connection whose Data map is nil --
// so the "state" assignment below it panicked with "assignment to entry in nil map".
func TestNewOAuthClient_EmptyProviderID(t *testing.T) {

	domainService := Domain{connectionService: &Connection{}}

	// An empty providerID is rejected by LoadOrCreateByProvider before it touches the session,
	// so a nil session is enough to reach the guard -- and is the proof it returns rather than
	// dereferencing anything.
	connection, err := domainService.NewOAuthClient(nil, "")

	require.Error(t, err)
	require.Equal(t, model.Connection{}, connection)
}

// TestDomain_Get_ZeroValue pins that a Domain service that has never loaded a record returns a
// blank Domain, and the SAME blank Domain on every call.
func TestDomain_Get_ZeroValue(t *testing.T) {

	service := Domain{}

	first := service.Get()
	require.NotNil(t, first)
	require.Equal(t, "default", first.ThemeID)
	require.NotNil(t, first.Connections)
	require.Same(t, first, service.Get())
}

// TestDomain_Publish pins that a published record replaces the cached one
func TestDomain_Publish(t *testing.T) {

	service := NewDomain()

	domain := model.NewDomain()
	domain.Label = "Published"
	service.publish(domain)

	require.Equal(t, "Published", service.Get().Label)
}

// TestDomain_Publish_KeepsOldSnapshots pins that publishing never modifies a record a reader
// already holds, which is what makes a background reload safe.
func TestDomain_Publish_KeepsOldSnapshots(t *testing.T) {

	service := NewDomain()

	before := model.NewDomain()
	before.Label = "Before"
	service.publish(before)

	held := service.Get()

	after := model.NewDomain()
	after.Label = "After"
	service.publish(after)

	require.Equal(t, "Before", held.Label)
	require.Equal(t, "After", service.Get().Label)
	require.NotSame(t, held, service.Get())
}

// TestDomain_Publish_Concurrent pins that readers and a background publisher can run at once.
// It proves nothing without -race.
func TestDomain_Publish_Concurrent(t *testing.T) {

	service := NewDomain()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for index := range 1000 {
			domain := model.NewDomain()
			domain.DatabaseVersion = uint(index)
			service.publish(domain)
		}
	}()

	for range 1000 {
		_ = service.Get().Label
	}

	<-done
	require.Equal(t, uint(999), service.Get().DatabaseVersion)
}

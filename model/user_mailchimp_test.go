package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testMasterKey is a 32-byte AES-256 key used only by tests in this file
var testMasterKey = []byte(testHex)

// testHex is filler shaped like the hex half of a credential, assembled from two
// halves so that secret scanners do not report this file as a live key
const testHex = "0123456789abcdef" + "0123456789abcdef"

// testMailchimpAPIKey is a syntactically valid Mailchimp API key built from that filler
const testMailchimpAPIKey = testHex + "-us6"

// testMailchimpWebhookSecret stands in for the secret Emissary mints per connection
const testMailchimpWebhookSecret = "a-webhook-secret"

// configuredUser returns a User with a complete, ACTIVE Mailchimp connection
func configuredUser(t *testing.T) User {

	t.Helper()

	user := NewUser()
	user.Vault.SetString(UserVaultMailchimpAPIKey, testMailchimpAPIKey)
	user.Vault.SetString(UserVaultMailchimpWebhookSecret, testMailchimpWebhookSecret)
	user.Data[UserDataMailchimpDataCenter] = "us6"
	user.Data[UserDataMailchimpAudience] = "abc123"
	user.Data[UserDataMailchimpTag] = "456"
	user.Data[UserDataMailchimpWebhook] = "789"
	user.Data[UserDataMailchimpState] = UserMailchimpStateActive

	return user
}

// TestUserMailchimp_IsConfigured confirms that a complete connection reads as
// configured, and that removing ANY single piece makes it read as incomplete
func TestUserMailchimp_IsConfigured(t *testing.T) {

	require.True(t, configuredUser(t).MailchimpIsConfigured())

	// Each case removes exactly one requirement from an otherwise complete connection
	removals := map[string]func(*User){
		"no api key":        func(u *User) { u.Vault.SetString(UserVaultMailchimpAPIKey, "") },
		"no webhook secret": func(u *User) { u.Vault.SetString(UserVaultMailchimpWebhookSecret, "") },
		"no data center":    func(u *User) { delete(u.Data, UserDataMailchimpDataCenter) },
		"no audience":       func(u *User) { delete(u.Data, UserDataMailchimpAudience) },
		"no tag":            func(u *User) { delete(u.Data, UserDataMailchimpTag) },
		"no webhook":        func(u *User) { delete(u.Data, UserDataMailchimpWebhook) },
	}

	for name, remove := range removals {

		t.Run(name, func(t *testing.T) {

			user := configuredUser(t)
			remove(&user)

			require.False(t, user.MailchimpIsConfigured(), "an incomplete connection must not read as configured")
			require.False(t, user.MailchimpIsActive(), "an incomplete connection can never be active")
		})
	}
}

// TestUserMailchimp_EmptyUser confirms that a User who has never touched Mailchimp reads
// as unconfigured rather than panicking on absent maps
func TestUserMailchimp_EmptyUser(t *testing.T) {

	user := NewUser()

	require.False(t, user.MailchimpIsConfigured())
	require.False(t, user.MailchimpIsActive())
	require.Equal(t, "", user.MailchimpState())
	require.Equal(t, "", user.MailchimpDataCenter())
	require.Equal(t, "", user.MailchimpAudienceID())
	require.Equal(t, "", user.MailchimpTagID())
	require.Equal(t, "", user.MailchimpWebhookID())
}

// TestUserMailchimp_ZeroValueUser confirms the accessors are safe on a bare struct,
// whose maps are nil because nothing ran NewUser
func TestUserMailchimp_ZeroValueUser(t *testing.T) {

	user := User{}

	require.False(t, user.MailchimpIsConfigured())
	require.False(t, user.MailchimpIsActive())
	require.Equal(t, "", user.MailchimpState())
}

// TestUserMailchimp_IsActive confirms that only the ACTIVE state permits a push
func TestUserMailchimp_IsActive(t *testing.T) {

	testCases := map[string]bool{
		UserMailchimpStateActive:    true,
		UserMailchimpStateReconnect: false,
		"":                          false,
		"SOMETHING-ELSE":            false,
	}

	for state, expected := range testCases {

		user := configuredUser(t)
		user.Data[UserDataMailchimpState] = state

		require.Equal(t, expected, user.MailchimpIsActive(), "state %q", state)
		require.True(t, user.MailchimpIsConfigured(), "state must not affect configuration")
	}
}

// TestUserMailchimp_SecretsDoNotExport confirms no Mailchimp secret reaches a User's
// data export
func TestUserMailchimp_SecretsDoNotExport(t *testing.T) {

	// ExportDocument marshals the whole struct with no filtering, so Vault's `json:"-"`
	// tags are the entire policy. Remove one and this fails instead of shipping a key.

	const apiKey = testMailchimpAPIKey
	const webhookSecret = testMailchimpWebhookSecret

	user := configuredUser(t)

	// Export BEFORE encrypting: plaintext lives in an unexported field, and this is the
	// state a User is in between a settings form and the database write.
	unsealed, err := json.Marshal(user)
	require.NoError(t, err)
	require.NotContains(t, string(unsealed), apiKey)
	require.NotContains(t, string(unsealed), webhookSecret)

	// Export AFTER encrypting: the ciphertext and its nonces must stay out too. They are
	// useless without the domain's master key, but the master key is one operator mistake
	// away and an export is a file that travels.
	require.NoError(t, user.Vault.Encrypt(testMasterKey))

	sealed, err := json.Marshal(user)
	require.NoError(t, err)
	require.NotContains(t, string(sealed), apiKey)
	require.NotContains(t, string(sealed), webhookSecret)
	require.NotContains(t, string(sealed), user.Vault.Encrypted[UserVaultMailchimpAPIKey])
	require.NotContains(t, strings.ToLower(string(sealed)), "\"vault\"")
	require.NotContains(t, strings.ToLower(string(sealed)), "\"nonce")

	// Round-trip the secret to prove the test is checking a Vault that actually held one,
	// rather than passing because nothing was ever stored.
	opened, err := user.Vault.Decrypt(testMasterKey)
	require.NoError(t, err)
	require.Equal(t, apiKey, opened[UserVaultMailchimpAPIKey])
	require.Equal(t, webhookSecret, opened[UserVaultMailchimpWebhookSecret])
}

// TestUserMailchimp_RemoteIDsDoExport confirms the non-secret half DOES export,
// because remote IDs are harmless in the User's own download
func TestUserMailchimp_RemoteIDsDoExport(t *testing.T) {

	user := configuredUser(t)

	result, err := json.Marshal(user)

	require.NoError(t, err)
	require.Contains(t, string(result), "abc123")
}

// TestUserMailchimp_KeysShareThePrefix pins the namespace that disconnect clears and
// the template sweep searches for
func TestUserMailchimp_KeysShareThePrefix(t *testing.T) {

	keys := []string{
		UserVaultMailchimpAPIKey,
		UserVaultMailchimpWebhookSecret,
		UserDataMailchimpDataCenter,
		UserDataMailchimpAudience,
		UserDataMailchimpTag,
		UserDataMailchimpWebhook,
		UserDataMailchimpState,
	}

	for _, key := range keys {
		require.True(t, strings.HasPrefix(key, UserMailchimpPrefix), "key %q must carry the Mailchimp prefix", key)
	}
}

// TestUserMailchimp_VaultIsObscuredThroughTheSchema confirms a secret reads back as
// the mask, not the value
func TestUserMailchimp_VaultIsObscuredThroughTheSchema(t *testing.T) {

	// This is what lets a settings form round-trip the field without erasing it. A
	// readable value here would put an API key into rendered HTML.

	user := configuredUser(t)
	value, ok := user.Vault.GetStringOK(UserVaultMailchimpAPIKey)

	require.True(t, ok)
	require.Equal(t, VaultObscuredValue, value)

	// Writing the mask back must not overwrite the stored secret
	user.Vault.SetString(UserVaultMailchimpAPIKey, VaultObscuredValue)

	require.NoError(t, user.Vault.Encrypt(testMasterKey))
	opened, err := user.Vault.Decrypt(testMasterKey)

	require.NoError(t, err)
	require.Equal(t, testMailchimpAPIKey, opened[UserVaultMailchimpAPIKey])
}

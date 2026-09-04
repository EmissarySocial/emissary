package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestVerifyWebhookSecret_MatchesTheStoredSecret confirms the happy path actually authorizes
func TestVerifyWebhookSecret_MatchesTheStoredSecret(t *testing.T) {

	service := UserConnection{encryptionKey: testDomainCipher}
	userConnection := readyConnection(t, "the-webhook-secret")

	require.True(t, service.VerifyWebhookSecret(&userConnection, "the-webhook-secret"))
}

// TestVerifyWebhookSecret_RejectsEverythingElse pins the authorization boundary of a public,
// unauthenticated route
func TestVerifyWebhookSecret_RejectsEverythingElse(t *testing.T) {

	service := UserConnection{encryptionKey: testDomainCipher}
	userConnection := readyConnection(t, "the-webhook-secret")

	rejected := map[string]string{
		"empty":                "",
		"wrong":                "not-the-webhook-secret",
		"a prefix of it":       "the-webhook-secre",
		"one character longer": "the-webhook-secrets",
		"case-shifted":         "The-Webhook-Secret",
	}

	for name, presented := range rejected {
		t.Run(name, func(t *testing.T) {
			require.False(t, service.VerifyWebhookSecret(&userConnection, presented), "must reject: %q", presented)
		})
	}
}

// TestVerifyWebhookSecret_EmptyStoredSecretNeverMatches guards the fall-through that would
// authorize every caller
func TestVerifyWebhookSecret_EmptyStoredSecretNeverMatches(t *testing.T) {

	// A connection whose secret was never minted stores "". ConstantTimeCompare("", "") is
	// TRUE, which would hand an unauthenticated caller every Follower the owner has. Two
	// redundant guards refuse it; this pins the property, which fails only if both go.

	service := UserConnection{encryptionKey: testDomainCipher}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.IsActive.Set(true)

	require.False(t, service.VerifyWebhookSecret(&userConnection, ""))
	require.False(t, service.VerifyWebhookSecret(&userConnection, "anything"))
}

// TestVerifyWebhookSecret_RequiresAReadyConnection confirms a webhook cannot outlive the
// connection that installed it
func TestVerifyWebhookSecret_RequiresAReadyConnection(t *testing.T) {

	// Mailchimp keeps firing at a URL until the webhook is deleted there, so a paused or
	// rejected connection must refuse rather than keep acting on the owner's followers.

	service := UserConnection{encryptionKey: testDomainCipher}

	paused := readyConnection(t, "the-webhook-secret")
	paused.IsActive.Set(false)
	require.False(t, service.VerifyWebhookSecret(&paused, "the-webhook-secret"), "a paused connection authorizes nothing")

	rejected := readyConnection(t, "the-webhook-secret")
	rejected.Status = model.UserConnectionStatusReconnect
	require.False(t, service.VerifyWebhookSecret(&rejected, "the-webhook-secret"), "a rejected connection authorizes nothing")
}

// TestMailchimpWebhook_UnregisteredEventsAreDiscarded confirms an event we did not ask for
// is not an error
func TestMailchimpWebhook_UnregisteredEventsAreDiscarded(t *testing.T) {

	// D11 registers `unsubscribe` and `upemail` only. Mailchimp sends what it sends, and a
	// `subscribe` delivery must never become a Follower (§5.6) -- so it is dropped, quietly.

	service := UserConnection{encryptionKey: testDomainCipher}
	userConnection := readyConnection(t, "the-webhook-secret")

	for _, event := range []string{"subscribe", "profile", "cleaned", "campaign", "", "BOGUS"} {
		require.NoError(t, service.Mailchimp_ReceiveWebhook(nil, &userConnection, event, nil), "event %q", event)
	}
}

// readyConnection returns a Mailchimp connection with a sealed webhook secret, as it would
// read after a round-trip through the database
func readyConnection(t *testing.T, secret string) model.UserConnection {

	t.Helper()

	encryptionKey, err := config.DecodeMasterKey(testDomainCipher)
	require.NoError(t, err)

	result := model.NewUserConnection()
	result.Type = model.UserConnectionTypeMailchimp
	result.IsActive.Set(true)
	result.Status = model.UserConnectionStatusReady
	result.Vault.SetString(model.UserConnectionVaultWebhookSecret, secret)

	require.NoError(t, result.Vault.Encrypt(encryptionKey))

	return result
}

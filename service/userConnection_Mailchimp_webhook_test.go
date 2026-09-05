package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// D11 registers `subscribe`, `unsubscribe`, and `upemail`. Mailchimp sends what it sends,
	// and a delivery we have no use for is dropped quietly rather than failed.

	service := UserConnection{encryptionKey: testDomainCipher}
	userConnection := readyConnection(t, "the-webhook-secret")

	for _, event := range []string{"profile", "cleaned", "campaign", "", "BOGUS"} {
		require.NoError(t, service.Mailchimp_ReceiveWebhook(nil, &userConnection, event, nil), "event %q", event)
	}
}

/******************************************
 * Inbound `subscribe` (D38)
 ******************************************/

// TestMailchimpSubscribe_CreatesAnActiveFollower walks the whole happy path of a member who
// joined the User's audience inside Mailchimp
func TestMailchimpSubscribe_CreatesAnActiveFollower(t *testing.T) {

	service, userConnection, session := newMailchimpWebhookService(t)

	err := service.Mailchimp_ReceiveWebhook(session, &userConnection, "subscribe", mapof.String{
		"type":                "subscribe",
		"data[email]":         "sarah@connor.mil",
		"data[merges][FNAME]": "Sarah",
		"data[merges][LNAME]": "Connor",
	})

	require.NoError(t, err)
	require.Len(t, session.collection.saved, 1)

	follower := session.collection.saved[0]

	require.Equal(t, model.FollowerStateActive, follower.StateID, "Mailchimp already did its own opt-in (D38)")
	require.Equal(t, model.FollowerMethodEmail, follower.Method)
	require.Equal(t, model.FollowerTypeUser, follower.ParentType)
	require.Equal(t, "sarah@connor.mil", follower.Actor.EmailAddress)
	require.Equal(t, "sarah@connor.mil", follower.Actor.ProfileURL, "LoadByActor matches on this field")
	require.Equal(t, "Sarah Connor", follower.Actor.Name)

	// Without a secret the unsubscribe link in every message Emissary is about to send this
	// person resolves to nothing, and they can never get off the list.
	require.NotEmpty(t, follower.Data.GetString("secret"))
}

// TestMailchimpSubscribe_PromotesAPendingFollower confirms that confirming at Mailchimp is
// accepted as confirmation
func TestMailchimpSubscribe_PromotesAPendingFollower(t *testing.T) {

	// Someone typed their address into Emissary's form, never clicked the link, and then
	// subscribed at Mailchimp instead.  That is consent arriving through the other door.

	pending := newEmailFollower(primitive.NilObjectID, "sarah@connor.mil")
	pending.StateID = model.FollowerStatePending

	service, userConnection, session := newMailchimpWebhookService(t, pending)

	err := service.Mailchimp_ReceiveWebhook(session, &userConnection, "subscribe", mapof.String{
		"data[email]": "sarah@connor.mil",
	})

	require.NoError(t, err)
	require.Len(t, session.collection.saved, 1)
	require.Equal(t, pending.FollowerID, session.collection.saved[0].FollowerID, "the existing record is promoted, not duplicated")
	require.Equal(t, model.FollowerStateActive, session.collection.saved[0].StateID)
}

// TestMailchimpSubscribe_NeverOverridesABlockRule is the guard that keeps an inbound webhook
// from undoing a decision the User made
func TestMailchimpSubscribe_NeverOverridesABlockRule(t *testing.T) {

	// A PAUSED Follower was paused by a BLOCK rule (D20).  Mailchimp knows nothing about that
	// rule and will keep reporting the member as subscribed, so an unguarded promotion would
	// quietly reverse the block -- and Emissary would resume emailing someone the User blocked.

	paused := newEmailFollower(primitive.NilObjectID, "sarah@connor.mil")
	paused.StateID = model.FollowerStatePaused

	service, userConnection, session := newMailchimpWebhookService(t, paused)

	err := service.Mailchimp_ReceiveWebhook(session, &userConnection, "subscribe", mapof.String{
		"data[email]": "sarah@connor.mil",
	})

	require.NoError(t, err, "a blocked member is not an error, just a no-op")
	require.Empty(t, session.collection.saved, "a blocked Follower must not be written at all")
}

// TestMailchimpSubscribe_LeavesAnActiveFollowerAlone confirms a duplicate delivery costs nothing
func TestMailchimpSubscribe_LeavesAnActiveFollowerAlone(t *testing.T) {

	// Mailchimp retries, and a re-save is not free once 1.2 hangs the outbound push on
	// Follower.Save: every duplicate would bounce the same member back at Mailchimp.

	active := newEmailFollower(primitive.NilObjectID, "sarah@connor.mil")
	active.StateID = model.FollowerStateActive

	service, userConnection, session := newMailchimpWebhookService(t, active)

	err := service.Mailchimp_ReceiveWebhook(session, &userConnection, "subscribe", mapof.String{
		"data[email]": "sarah@connor.mil",
	})

	require.NoError(t, err)
	require.Empty(t, session.collection.saved)
}

// TestMailchimpSubscribe_DiscardsPayloadsThatCannotBecomeFollowers pins the difference between
// a bad payload and a broken server
func TestMailchimpSubscribe_DiscardsPayloadsThatCannotBecomeFollowers(t *testing.T) {

	// Every value here is attacker-supplied.  Returning an error would answer 500 and roll the
	// transaction back, and Mailchimp would retry that same delivery for as long as the webhook
	// is installed -- so a payload that cannot make a valid Follower is dropped instead.

	payloads := map[string]mapof.String{
		"no address":       {"data[email]": ""},
		"only whitespace":  {"data[email]": "   "},
		"not an address":   {"data[email]": "not-an-email-address"},
		"address too long": {"data[email]": strings.Repeat("a", 200) + "@connor.mil"},
		"name too long":    {"data[email]": "sarah@connor.mil", "data[merges][FNAME]": strings.Repeat("Sarah ", 40)},
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {

			service, userConnection, session := newMailchimpWebhookService(t)

			require.NoError(t, service.Mailchimp_ReceiveWebhook(session, &userConnection, "subscribe", payload))
			require.Empty(t, session.collection.saved)
		})
	}
}

// TestMailchimpMemberName verifies that FNAME and LNAME rejoin the way D8 split them
func TestMailchimpMemberName(t *testing.T) {

	tests := map[string]struct {
		firstName string
		lastName  string
		expected  string
	}{
		"both names":      {"Sarah", "Connor", "Sarah Connor"},
		"first only":      {"Sarah", "", "Sarah"},
		"last only":       {"", "Connor", "Connor"},
		"neither":         {"", "", ""},
		"padded":          {"  Sarah  ", "  Connor  ", "Sarah Connor"},
		"only whitespace": {"   ", "   ", ""},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := mailchimpMemberName(mapof.String{
				"data[merges][FNAME]": test.firstName,
				"data[merges][LNAME]": test.lastName,
			})

			require.Equal(t, test.expected, result)
		})
	}
}

// TestMailchimpNewFollower_MintsAUniqueSecret confirms two subscribers never share an
// unsubscribe link
func TestMailchimpNewFollower_MintsAUniqueSecret(t *testing.T) {

	first, err := mailchimpNewFollower(primitive.NilObjectID, "sarah@connor.mil", nil)
	require.NoError(t, err)

	second, err := mailchimpNewFollower(primitive.NilObjectID, "sarah@connor.mil", nil)
	require.NoError(t, err)

	require.NotEqual(t, first.Data.GetString("secret"), second.Data.GetString("secret"))
}

// newMailchimpWebhookService returns a UserConnection service whose Followers live in memory,
// along with the connection that owns them
func newMailchimpWebhookService(t *testing.T, followers ...model.Follower) (UserConnection, model.UserConnection, followerSession) {

	t.Helper()

	// The owner's ID is deliberately zero: Follower.Save recalculates the owner's follower
	// count, and CalcFollowerCount short-circuits on a zero ID rather than reaching for a User
	// collection this harness does not have.
	followerService, session := newFollowerService(followers...)
	userConnection := readyConnection(t, "the-webhook-secret")

	return UserConnection{encryptionKey: testDomainCipher, followerService: followerService}, userConnection, session
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

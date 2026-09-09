package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

/******************************************
 * Mailchimp Connection
 *
 * Setup for a User's own Mailchimp account. The credential is checked, proven
 * against the Mailchimp API, and only then written -- so a key the service
 * refuses never reaches the database. Nothing here parses a credential, which
 * is why the data center is its own configured value (D34).
 ******************************************/

// mailchimpWebhookSecretLength is the length of the secret that authenticates inbound
// webhook deliveries from Mailchimp
const mailchimpWebhookSecretLength = 48

// connect installs this connection at its remote service, or removes it, following the
// User's own IsActive switch
func (service *UserConnection) connect(session data.Session, userConnection *model.UserConnection) error {

	const location = "service.UserConnection.connect"

	// A connection that was already off has nothing installed to remove
	if userConnection.IsActive.IsFalse() {

		if userConnection.IsActive.NotChanged() {
			return nil
		}

		return service.disconnect(session, userConnection)
	}

	// An active connection is re-proved on every save, deliberately: Save is reached only from
	// the settings form, and re-running setup is what puts back a webhook the User deleted at
	// Mailchimp. Setup is idempotent, so an unchanged save writes nothing (see AGENTS.md).

	switch userConnection.Type {

	case model.UserConnectionTypeMailchimp:
		return service.mailchimp_connect(userConnection)
	}

	return derp.Internal(location, "Unrecognized connection type", userConnection.Type)
}

// disconnect removes this connection from its remote service, leaving the credential in place
func (service *UserConnection) disconnect(session data.Session, userConnection *model.UserConnection) error {

	const location = "service.UserConnection.disconnect"

	switch userConnection.Type {

	case model.UserConnectionTypeMailchimp:
		return service.mailchimp_disconnect(session, userConnection)
	}

	return derp.Internal(location, "Unrecognized connection type", userConnection.Type)
}

// mailchimp_connect proves this connection's credential against the Mailchimp API and
// marks it ready to use
func (service *UserConnection) mailchimp_connect(userConnection *model.UserConnection) error {

	const location = "service.UserConnection.mailchimp_connect"

	// Resolve the credential the User is asking us to use
	apiKey, err := service.mailchimp_apiKey(userConnection)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read your saved Mailchimp API key")
	}

	dataCenter := strings.TrimSpace(userConnection.Data.GetString(model.UserConnectionDataCenter))

	// RULE: both values must be storable and sendable before either reaches the network.
	// These errors are written for the User, so they travel to the form unwrapped.
	if err := mailchimp.ValidateAPIKey(apiKey); err != nil {
		return err
	}

	if err := mailchimp.ValidateDataCenter(dataCenter); err != nil {
		return err
	}

	client, err := mailchimp.New(apiKey, dataCenter)

	if err != nil {
		return derp.Wrap(err, location, "Unable to reach Mailchimp with these values")
	}

	audienceID := strings.TrimSpace(userConnection.Data.GetString(model.UserConnectionDataAudienceID))

	// RULE: Mailchimp is the only authority on whether these values work, and the Audience ID
	// is as opaque as the credential is (D34) -- so it is proven by using it, not by parsing
	// it. Reading the chosen audience proves the key, the data center, AND the ID at once.
	audience, err := service.mailchimp_verifyAudience(client, audienceID)

	if err != nil {
		return err
	}

	// RULE: mint the webhook secret once and never again. Rotating it on a re-save would
	// orphan the webhook already installed at Mailchimp with the previous value.
	if !userConnection.Vault.HasString(model.UserConnectionVaultWebhookSecret) {

		secret, err := random.GenerateString(mailchimpWebhookSecretLength)

		if err != nil {
			return derp.Wrap(err, location, "Unable to generate a webhook secret")
		}

		userConnection.Vault.SetString(model.UserConnectionVaultWebhookSecret, secret)
	}

	// Write the credential back in its resolved form, so a mask never reaches the database
	userConnection.Vault.SetString(model.UserConnectionVaultAPIKey, apiKey)
	userConnection.Data.SetString(model.UserConnectionDataCenter, dataCenter)

	// RULE: a proven credential is NOT a finished connection (D39). Without an audience there
	// is nowhere to sync to, so the connection stays PENDING -- visible as half-configured
	// rather than trusted by everything downstream.
	if audienceID == "" {
		userConnection.Status = model.UserConnectionStatusPending
		return nil
	}

	// Connected. Now go set up the audience.
	return service.mailchimp_setup(client, userConnection, audience)
}

// mailchimp_verifyAudience reads the audience a User pasted the ID of, or reports an empty
// audience as the half-finished state it is
func (service *UserConnection) mailchimp_verifyAudience(client mailchimp.Client, audienceID string) (mailchimp.Audience, error) {

	// An empty ID is not an error: the credential still has to be proven, so that the User
	// can save a key now and paste an Audience ID later. Ping proves it and fetches nothing.
	if audienceID == "" {

		if err := client.Ping(); err != nil {
			return mailchimp.Audience{}, err
		}

		return mailchimp.Audience{}, nil
	}

	return client.GetAudience(audienceID)
}

// mailchimp_disconnect removes what Emissary installed in the User's Mailchimp account,
// leaving their credential and their members alone
func (service *UserConnection) mailchimp_disconnect(_ data.Session, userConnection *model.UserConnection) error {

	const location = "service.UserConnection.mailchimp_disconnect"

	// RULE: report a failed teardown, never propagate it (D37). A User whose key was revoked,
	// or whose Mailchimp is unreachable, must still be able to switch the connection off --
	// refusing here would strand them with one that cannot be paused. Delete does the same.
	if err := service.mailchimp_removeWebhook(userConnection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to remove the webhook from your Mailchimp account", userConnection.UserConnectionID))
	}

	// Turning the connection off stops the sync immediately, because every hook reads
	// IsReady() rather than the presence of a credential.
	userConnection.Status = model.UserConnectionStatusPending
	userConnection.Data.Remove(model.UserConnectionDataWebhookID)

	return nil
}

// mailchimp_apiKey returns the credential to verify: the one the User just typed, or the
// one already in the Vault when the form posted the mask back
func (service *UserConnection) mailchimp_apiKey(userConnection *model.UserConnection) (string, error) {

	const location = "service.UserConnection.mailchimp_apiKey"

	// RULE: Vault.GetStringOK hands a form the mask and never the secret, so a value coming
	// back means "no change" unless it is a real one. SetString has already dropped the mask.
	vault, err := service.DecryptVault(userConnection, model.UserConnectionVaultAPIKey)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to open your saved secrets")
	}

	return strings.TrimSpace(vault.GetString(model.UserConnectionVaultAPIKey)), nil
}

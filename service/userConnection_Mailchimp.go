package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
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

	// RULE: nothing to do unless the switch moved or a new secret was entered. Save() runs on
	// every edit, and re-proving an unchanged connection would put a remote API call behind
	// routine writes.
	if userConnection.IsActive.NotChanged() && !userConnection.Vault.NeedsEncryption() {
		return nil
	}

	// A connection that was already off has nothing installed to remove
	if userConnection.IsActive.IsFalse() {

		if userConnection.IsActive.NotChanged() {
			return nil
		}

		return service.disconnect(session, userConnection)
	}

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

	// RULE: Mailchimp is the only authority on whether the pair works. A 401 means the key,
	// a 404 usually means the data center, and a success returns the audiences to choose from.
	if _, err := service.mailchimp_audiences(apiKey, dataCenter); err != nil {
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
	userConnection.Status = model.UserConnectionStatusReady

	// Connected. Now go pick an audience.
	return nil
}

// mailchimp_disconnect removes what Emissary installed in the User's Mailchimp account,
// leaving their credential and their members alone
func (service *UserConnection) mailchimp_disconnect(_ data.Session, userConnection *model.UserConnection) error {

	// Turning the connection off stops the sync immediately, because every hook reads
	// IsReady() rather than the presence of a credential.
	userConnection.Status = ""

	// TODO(MAILING-LISTS.md 1.1): delete the installed webhook once one is installed. There
	// is nothing at Mailchimp to remove until audience setup runs.
	userConnection.Data.Remove(model.UserConnectionDataWebhookID)

	return nil
}

// mailchimp_audiences reads the audiences that a credential can reach
func (service *UserConnection) mailchimp_audiences(apiKey string, dataCenter string) (sliceof.Object[mailchimp.Audience], error) {

	const location = "service.UserConnection.mailchimp_audiences"

	client, err := mailchimp.New(apiKey, dataCenter)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to reach Mailchimp with these values")
	}

	audiences, err := client.GetAudiences()

	if err != nil {
		return nil, err
	}

	return audiences, nil
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

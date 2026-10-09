package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
)

/******************************************
 * Mailchimp Audience Setup
 *
 * What Emissary installs inside the audience a User names: a webhook, to
 * hear about changes made at Mailchimp. It is create-if-missing, because
 * this runs again on every reconnect. See MAILING-LISTS.md 1.1.
 ******************************************/

// mailchimp_setup installs everything Emissary needs inside the audience this User chose
func (service *UserConnection) mailchimp_setup(client mailchimp.Client, userConnection *model.UserConnection, audience mailchimp.Audience) error {

	const location = "service.UserConnection.mailchimp_setup"

	audienceID := audience.ID

	// Record the audience. Its NAME is stored so that the settings page can show the User
	// which list they actually pasted the ID of -- the ID alone is unreadable, which is how
	// a wrong paste stays wrong.
	userConnection.Data.SetString(model.UserConnectionDataAudienceID, audienceID)
	userConnection.Data.SetString(model.UserConnectionDataAudienceName, audience.Name)

	// RULE: never ask Mailchimp to deliver to a local domain (D45). Nothing on the public
	// internet can reach one, so the install could only fail -- and on a development machine
	// that is the ordinary case. This is the ONE place a connection is READY with no webhook.
	if uri.IsLocalHostname(uri.Hostname(service.host)) {
		log.Debug().Str("host", service.host).Msg("Mailchimp webhook not installed: local domain")
		userConnection.Status = model.UserConnectionStatusReady
		return nil
	}

	// RULE: on a public domain the webhook is part of being set up (D47). A connection whose
	// webhook did not install is not READY, and the User sees why -- a silent half-connection
	// would push outward while every unsubscribe made at Mailchimp quietly never arrived.
	webhookID, err := service.mailchimp_setupWebhook(client, userConnection, audienceID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to install a Mailchimp webhook")
	}

	// Set up, and ready to use (D39)
	userConnection.Data.SetString(model.UserConnectionDataWebhookID, webhookID)
	userConnection.Status = model.UserConnectionStatusReady

	return nil
}

// mailchimp_setupWebhook returns the ID of the webhook that delivers this audience's events
// to Emissary, installing one if it is absent
func (service *UserConnection) mailchimp_setupWebhook(client mailchimp.Client, userConnection *model.UserConnection, audienceID string) (string, error) {

	const location = "service.UserConnection.mailchimp_setupWebhook"

	callbackURL, err := service.mailchimp_callbackURL(userConnection)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to build this connection's callback address")
	}

	webhooks, err := client.GetWebhooks(audienceID)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to read this audience's webhooks")
	}

	// RULE: list first, then create. This is the ONE call in setup that is not idempotent --
	// a second webhook on the same URL makes Mailchimp deliver every event twice, which is
	// two passes through Follower.Save per subscribe, forever, and nothing reports it.
	for _, webhook := range webhooks {
		if webhook.URL == callbackURL {
			return webhook.ID, nil
		}
	}

	events := mailchimp.WebhookEvents{
		Subscribe:   true,
		Unsubscribe: true,
		UpEmail:     true,
	}

	// RULE: `user` and `admin`, never `api` (D12). Mailchimp fires webhooks for API-initiated
	// changes too, so registering `api` would echo every member Emissary adds straight back
	// at Emissary's own handler.
	sources := mailchimp.WebhookSources{
		User:  true,
		Admin: true,
	}

	webhook, err := client.CreateWebhook(audienceID, callbackURL, events, sources)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to install a webhook")
	}

	return webhook.ID, nil
}

// mailchimp_callbackURL returns the address Mailchimp delivers this connection's events to
func (service *UserConnection) mailchimp_callbackURL(userConnection *model.UserConnection) (string, error) {

	const location = "service.UserConnection.mailchimp_callbackURL"

	vault, err := service.DecryptVault(userConnection, model.UserConnectionVaultWebhookSecret)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to open your saved secrets")
	}

	secret := vault.GetString(model.UserConnectionVaultWebhookSecret)

	// RULE: refuse to install a webhook that nothing can authorize. An empty secret would
	// publish an unauthenticated callback address to a third party, and every delivery it
	// carried would be rejected anyway.
	if secret == "" {
		return "", derp.Internal(location, "Webhook secret has not been minted", userConnection.UserConnectionID)
	}

	// The secret rides in the query string because that is the only part of a Mailchimp
	// callback address we control (D25).
	return service.host + "/.mailchimp/webhook/" + userConnection.UserConnectionID.Hex() + "?secret=" + url.QueryEscape(secret), nil
}

// mailchimp_removeWebhook deletes the webhook Emissary installed, if there is one to delete
func (service *UserConnection) mailchimp_removeWebhook(userConnection *model.UserConnection) error {

	const location = "service.UserConnection.mailchimp_removeWebhook"

	webhookID := userConnection.Data.GetString(model.UserConnectionDataWebhookID)
	audienceID := userConnection.Data.GetString(model.UserConnectionDataAudienceID)

	// RULE: no webhook ID means no webhook was ever installed -- D45's local-domain skip
	// leaves exactly this -- so there is nothing at Mailchimp to remove.
	if webhookID == "" {
		return nil
	}

	// RULE: no audience ID means setup never reached an audience (D39), so nothing could have
	// been installed under one, and there is no path to address a delete to anyway.
	if audienceID == "" {
		return nil
	}

	apiKey, err := service.mailchimp_apiKey(userConnection)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read your saved Mailchimp API key")
	}

	client, err := mailchimp.New(apiKey, userConnection.Data.GetString(model.UserConnectionDataCenter))

	if err != nil {
		return derp.Wrap(err, location, "Unable to reach Mailchimp")
	}

	// DeleteWebhook treats a webhook that is already gone as success, which is what lets a
	// User who deleted it by hand still disconnect (D37).
	return client.DeleteWebhook(audienceID, webhookID)
}

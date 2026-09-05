package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

/******************************************
 * Mailchimp Audience Setup
 *
 * What Emissary installs inside the audience a User names: a merge field to
 * carry the FollowerID, and a webhook to hear about changes made at Mailchimp.
 * Both are create-if-missing, because this runs again on every reconnect.
 * See MAILING-LISTS.md 1.1.
 ******************************************/

// mailchimpMergeFieldTag is the merge field carrying the Emissary FollowerID (D8).
// Mailchimp caps a merge tag at 10 characters, and this is exactly 10.
const mailchimpMergeFieldTag = "EMISSARYID"

// mailchimpMergeFieldName is the human-readable label shown beside that merge field
const mailchimpMergeFieldName = "Emissary ID"

// mailchimp_setup installs everything Emissary needs inside the audience this User chose
func (service *UserConnection) mailchimp_setup(client mailchimp.Client, userConnection *model.UserConnection, audience mailchimp.Audience) error {

	const location = "service.UserConnection.mailchimp_setup"

	audienceID := audience.ID

	// Find or create the EMISSARYID merge field (D8)
	if err := mailchimp_setupMergeField(client, audienceID); err != nil {
		return derp.Wrap(err, location, "Unable to set up the EMISSARYID field in this audience")
	}

	// Record what was installed, and mark the connection ready to use (D39). The audience NAME
	// is stored so that the settings page can show the User which list they actually pasted
	// the ID of -- the ID alone is unreadable, which is how a wrong paste stays wrong.
	userConnection.Data.SetString(model.UserConnectionDataAudienceID, audienceID)
	userConnection.Data.SetString(model.UserConnectionDataAudienceName, audience.Name)
	userConnection.Status = model.UserConnectionStatusReady

	// RULE: a webhook that will not install does NOT block the connection (D40). It is the
	// only step that asks Mailchimp to reach back at this server -- which a development
	// machine cannot receive -- so it is reported, and its absence recorded, rather than
	// refused. Outbound sync works without it; only inbound events are missing.
	webhookID, err := service.mailchimp_setupWebhook(client, userConnection, audienceID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to install a Mailchimp webhook", userConnection.UserConnectionID))
		return nil
	}

	userConnection.Data.SetString(model.UserConnectionDataWebhookID, webhookID)

	return nil
}

// mailchimp_setupMergeField creates this audience's EMISSARYID merge field if it is absent
func mailchimp_setupMergeField(client mailchimp.Client, audienceID string) error {

	const location = "service.mailchimp_setupMergeField"

	mergeFields, err := client.GetMergeFields(audienceID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read this audience's fields")
	}

	// Members are addressed by merge TAG rather than by ID, so nothing needs to be recorded
	// here -- only that the field exists.
	for _, mergeField := range mergeFields {
		if mergeField.Tag == mailchimpMergeFieldTag {
			return nil
		}
	}

	if _, err := client.CreateMergeField(audienceID, mailchimpMergeFieldTag, mailchimpMergeFieldName); err != nil {
		return derp.Wrap(err, location, "Unable to create the EMISSARYID field. An audience can hold only 30 custom fields.")
	}

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

	// RULE: refuse a callback Mailchimp could never deliver to, BEFORE asking it to try. A
	// development machine is the ordinary case here -- nothing on the public internet can
	// reach `localhost` -- and failing fast says so, where failing at Mailchimp does not.
	// This is not fatal: D40 records the miss as an empty webhookId and the connection works
	// outbound regardless.
	if uri.IsLocalURL(callbackURL) {
		return "", derp.BadRequest(location, "Mailchimp cannot deliver to this server's address, so no webhook was installed. Outbound syncing still works; inbound changes made at Mailchimp will not arrive here.", uri.Hostname(service.host))
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

	// A connection that never finished setup has nothing installed to remove
	if (webhookID == "") || (audienceID == "") {
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

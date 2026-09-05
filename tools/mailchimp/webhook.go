package mailchimp

import (
	"github.com/benpate/rosetta/sliceof"
)

// Webhook is one callback that Mailchimp delivers audience events to
type Webhook struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// WebhookEvents names the audience events a webhook is delivered for
type WebhookEvents struct {
	Subscribe   bool `json:"subscribe"`
	Unsubscribe bool `json:"unsubscribe"`
	Profile     bool `json:"profile"`
	Cleaned     bool `json:"cleaned"`
	UpEmail     bool `json:"upemail"`
	Campaign    bool `json:"campaign"`
}

// WebhookSources names where a change has to originate for a webhook to fire
type WebhookSources struct {
	User  bool `json:"user"`
	Admin bool `json:"admin"`
	API   bool `json:"api"`
}

// GetWebhooks returns every webhook installed on the provided audience
func (client Client) GetWebhooks(audienceID string) (sliceof.Object[Webhook], error) {

	const location = "tools.mailchimp.Client.GetWebhooks"

	var result struct {
		Webhooks sliceof.Object[Webhook] `json:"webhooks"`
	}

	transaction := client.get("/lists/"+audienceID+"/webhooks").
		Query("fields", "webhooks.id,webhooks.url").
		Result(&result)

	if err := transaction.Send(); err != nil {
		return nil, describeError(err, location, "Unable to read webhooks from Mailchimp")
	}

	return result.Webhooks, nil
}

// CreateWebhook installs a webhook that delivers the provided events to the provided URL
func (client Client) CreateWebhook(audienceID string, callbackURL string, events WebhookEvents, sources WebhookSources) (Webhook, error) {

	const location = "tools.mailchimp.Client.CreateWebhook"

	result := Webhook{}

	body := map[string]any{
		"url":     callbackURL,
		"events":  events,
		"sources": sources,
	}

	transaction := client.post("/lists/"+audienceID+"/webhooks", body).Result(&result)

	if err := transaction.Send(); err != nil {
		return Webhook{}, describeError(err, location, "Unable to install a webhook in Mailchimp")
	}

	return result, nil
}

// DeleteWebhook removes an installed webhook, and treats one that is already gone as success
func (client Client) DeleteWebhook(audienceID string, webhookID string) error {

	const location = "tools.mailchimp.Client.DeleteWebhook"

	transaction := client.delete("/lists/" + audienceID + "/webhooks/" + webhookID)

	if err := transaction.Send(); err != nil {

		// RULE: a webhook that is already gone is the outcome this method exists to reach.
		// The User can delete one by hand inside Mailchimp, and refusing to disconnect
		// because there was nothing to disconnect is the worst possible time to refuse.
		if isNotFound(err) {
			return nil
		}

		return describeError(err, location, "Unable to remove a webhook from Mailchimp")
	}

	return nil
}

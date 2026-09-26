package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// StripeConnectNeedsRepair returns TRUE if this Domain's Stripe Connect Connection is active on a
// public hostname but has no stored webhook signing secret.
func (service *Connection) StripeConnectNeedsRepair() bool {

	connection, exists := service.connections()[model.ConnectionProviderStripeConnect]

	if !exists {
		return false
	}

	if !connection.Active {
		return false
	}

	// RULE: A local hostname never registers a webhook, so there is nothing to repair
	if uri.IsLocalHostname(service.host) {
		return false
	}

	return !connection.Vault.HasString("webhookSecret")
}

// RepairStripeConnect replaces the webhook endpoint of a Stripe Connect Connection whose signing
// secret was never stored, and returns TRUE if it did.
func (service *Connection) RepairStripeConnect(session data.Session) (bool, error) {

	const location = "service.Connection.RepairStripeConnect"

	// RULE: Check again, because another server may have repaired it since this was queued
	if !service.StripeConnectNeedsRepair() {
		return false, nil
	}

	// Load the Connection
	connection := model.NewConnection()

	if err := service.LoadByProvider(session, model.ConnectionProviderStripeConnect, &connection); err != nil {
		return false, derp.Wrap(err, location, "Loading Stripe Connect Connection")
	}

	// Saving an active Connection runs StripeConnect.Connect, which replaces the endpoint
	if err := service.Save(session, &connection, "Replaced a webhook endpoint whose secret was never stored"); err != nil {
		return false, derp.Wrap(err, location, "Saving Stripe Connect Connection")
	}

	// Phoenix, meet ashes
	return true, nil
}

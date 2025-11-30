package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

// Connection represents a single connection to an individual Provider.  It usually contains an OAuth2 token, but may also contain
// other connection information like a username or password.  It may also represent a connection that is still being formed,
// for instance, storing the intermediate state of an OAuth2 connection that has not yet completed the three-legged handshake.
type Connection struct {
	ConnectionID primitive.ObjectID `bson:"_id"`        // Unique ID for this connection
	ProviderID   string             `bson:"providerId"` // ID of the provider that this credential accesses
	Type         string             `bson:"type"`       // Type of connection (e.g. "payment")
	Data         mapof.String       `bson:"data"`       // Unique data for this credential
	Vault        Vault              `bson:"vault"`      // Secure secret storage for this connection
	Token        *oauth2.Token      `bson:"token"`      // OAuth2 Token (if necessary)
	Active       bool               `bson:"active"`     // Is this credential active?

	journal.Journal `json:"-" bson:",inline"`
}

// NewConnection returns a fully initialized Connection object.
func NewConnection() Connection {
	return Connection{
		ConnectionID: primitive.NewObjectID(),
		Data:         mapof.NewString(),
		Vault:        NewVault(),
	}
}

// ID implements the set.Value interface
func (connection Connection) ID() string {
	return connection.ConnectionID.Hex()
}

func (connection Connection) LookupCode() form.LookupCode {

	switch connection.ProviderID {

	case ConnectionProviderStripe:
		return form.LookupCode{
			Group:       "MANUAL",
			Value:       connection.ProviderID,
			Label:       "Stripe",
			Description: "Stripe is a sophisticated payment platform for techies. Manage your own Stripe API keys.",
			Icon:        "/.templates/user-settings/resources/stripe.svg",
		}

	case ConnectionProviderStripeConnect:
		return form.LookupCode{
			Group:       "OAUTH",
			Value:       connection.ProviderID,
			Label:       "Stripe Connect",
			Description: "Stripe Connect is a powerful payment platform for techies. Connect your Stripe account via OAuth.",
			Icon:        "/.templates/user-settings/resources/stripe.svg",
		}

		// case ConnectionProviderPayPal:
		//	return form.LookupCode{
		//		Group:       "MANUAL",
		//		Value:       connection.ProviderID,
		//		Label:       "PayPal",
		//		Description: "PayPal is a leading payment platform for consumers and small businesses.",
		//		Icon:        "/.templates/user-settings/resources/paypal.png",
		//	}
	}

	return form.LookupCode{
		Value: connection.ProviderID,
		Label: connection.ProviderID,
	}
}

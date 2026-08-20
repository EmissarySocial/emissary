package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"golang.org/x/oauth2"
)

// Provider is the lifecycle that every third-party Connection follows
type Provider interface {

	// Lifecycle Methods
	BeforeSave(connection *model.Connection, vault mapof.String) error
	Connect(connection *model.Connection, vault mapof.String, host string) error
	Refresh(connection *model.Connection, vault mapof.String) error
	Disconnect(connection *model.Connection, vault mapof.String) error
}

// OAuthProvider is implemented by Providers that authenticate through OAuth
type OAuthProvider interface {
	OAuthConfig() oauth2.Config
}

// ManualProvider is implemented by Providers that are configured by hand, using a form
type ManualProvider interface {
	ManualConfig() form.Form
}

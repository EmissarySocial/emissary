package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"golang.org/x/oauth2"
)

type Provider interface {

	// Lifecycle Methods
	Connect(connection *model.Connection, vault mapof.String, host string) error
	Refresh(connection *model.Connection, vault mapof.String) error
	Disconnect(connection *model.Connection, vault mapof.String) error
}

type OAuthProvider interface {
	OAuthConfig() oauth2.Config
}

type ManualProvider interface {
	ManualConfig() form.Form
}

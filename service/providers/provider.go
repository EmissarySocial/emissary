package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"golang.org/x/oauth2"
)

type Provider interface {

	// Lifecycle Methods
	AfterConnect(factory Factory, client *model.Connection) error
	AfterUpdate(factory Factory, client *model.Connection) error
}

type OAuthProvider interface {
	OAuthConfig() oauth2.Config
}

type ManualProvider interface {
	ManualConfig() form.Form
}

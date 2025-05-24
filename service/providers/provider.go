package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"golang.org/x/oauth2"
)

type Provider interface {

	// Lifecycle Methods
	AfterConnect(factory Factory, client *model.Connection, vault mapof.String) error
	AfterUpdate(factory Factory, client *model.Connection, vault mapof.String) error
}

type OAuthProvider interface {
	OAuthConfig() oauth2.Config
}

type ManualProvider interface {
	ManualConfig() form.Form
}

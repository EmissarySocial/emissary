package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"golang.org/x/oauth2"
)

type Provider interface {

	// Lifecycle Methods
	AfterConnect(factory Factory, client *model.Client) error
	AfterUpdate(factory Factory, client *model.Client) error

	// Provider Methods
	PollStreams(client *model.Client) <-chan model.Stream
}

type OAuthProvider interface {
	OAuthConfig() oauth2.Config
}

type ManualProvider interface {
	ManualConfig() form.Form
}

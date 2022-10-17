package external

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"golang.org/x/oauth2"
)

type Adapter interface {
	PollStreams(model.Client) error
	PostStream(model.Client) error
}

type Installer interface {
	Install(Factory, *model.Client) error
}

type OAuthAdapter interface {
	OAuthConfig() oauth2.Config
}

type ManualAdapter interface {
	ManualConfig() form.Form
}

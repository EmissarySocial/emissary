package external

import (
	"github.com/benpate/form"
	"golang.org/x/oauth2"
)

type Adapter interface {
	Install()
	PollStreams()
	PostStream()
}

type OAuthAdapter interface {
	OAuthConfig() oauth2.Config
}

type ManualAdapter interface {
	ManualConfig() form.Form
}

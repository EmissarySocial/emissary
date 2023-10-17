package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
)

func Mastodon(serverFactory *server.Factory) toot.API {

	return toot.API{
		Authorize:                        mastodon_Authorize(serverFactory),
		PostApplication:                  mastodon_PostApplication(serverFactory),
		GetApplication_VerifyCredentials: mastodon_GetApplication_VerifyCredentials(serverFactory),
	}
}

func mastodon_Authorize(serverFactory *server.Factory) func(*http.Request, ...string) bool {

	return func(request *http.Request, scopes ...string) bool {
		return true
	}
}

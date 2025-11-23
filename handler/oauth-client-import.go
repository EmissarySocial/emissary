package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func GetOAuthImportCallback(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.OAuthImportCallback"

	// Collect URL arguents
	providerID := ctx.Param("provider")
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")

	// Load the currently active Import record
	importService := factory.Import()
	record := model.NewImport()

	if err := importService.LoadByToken(session, user.UserID, state, &record); err != nil {
		return derp.Wrap(err, location, "Unable to load corresponding import record", user.UserID, state)
	}

	// If this record is still "Authorizing", then exchange the OAuth code for a real OAuth token
	if record.StateID == model.ImportStateAuthorizing {

		if err := importService.OAuthExchange(session, &record, state, code); err != nil {
			return derp.Wrap(err, location, "Unable to exchange code for token", providerID, code)
		}
	}

	// Continue with import UX
	return ctx.Redirect(http.StatusTemporaryRedirect, "/@me/settings/import")
}

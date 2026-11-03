package mastodon

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// PostReport refuses a report with 422
// https://docs.joinmastodon.org/methods/reports/
func PostReport(serverFactory *server.Factory) func(model.Authorization, txn.PostReport) (object.Report, error) {

	// RULE: Emissary has nowhere to send a report yet, so this says so instead of pretending it was filed.

	return func(model.Authorization, txn.PostReport) (object.Report, error) {
		return object.Report{}, derp.BadRequest("handler.mastodon.PostReport", "Reports are not supported on this server yet", derp.WithCode(http.StatusUnprocessableEntity))
	}
}

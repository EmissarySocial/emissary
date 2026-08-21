package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// PostReport handles POST /api/v1/reports
// https://docs.joinmastodon.org/methods/reports/
func PostReport(serverFactory *server.Factory) func(model.Authorization, txn.PostReport) (object.Report, error) {

	const location = "handler.mastodon.PostReport"

	return func(auth model.Authorization, report txn.PostReport) (object.Report, error) {

		factory, err := serverFactory.ByHostname(report.Host)

		if err != nil {
			return object.Report{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		return factory.Moderation().SubmitReport(auth, report)
	}
}

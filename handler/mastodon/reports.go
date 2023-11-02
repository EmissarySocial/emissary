package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/reports/
func PostReport(serverFactory *server.Factory) func(model.Authorization, txn.PostReport) (object.Report, error) {

	return func(model.Authorization, txn.PostReport) (object.Report, error) {
		return object.Report{}, nil
	}
}

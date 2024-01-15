package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/mutes/
func GetMutes(serverFactory *server.Factory) func(model.Authorization, txn.GetMutes) ([]object.Account, toot.PageInfo, error) {

	// const location = "handler.mastodon.GetMutes"

	return func(auth model.Authorization, t txn.GetMutes) ([]object.Account, toot.PageInfo, error) {

		/*
			// Get the factory for this Domain
			factory, err := serverFactory.ByDomainName(t.Host)

			if err != nil {
				return nil, derp.Wrap(err, location, "Invalid Domain")
			}

			ruleService := factory.Rule()

			// Locate Rule for the Current User
			rules, err := ruleService.QueryActiveByUser(auth.UserID, model.RuleTypeActor)

			if err != nil {
				return nil, derp.Wrap(err, location, "Error querying rules")
			}

			return getSliceOfToots[model.Rule, object.Account](rules), getPageInfo(rules), nil
		*/
		return []object.Account{}, toot.PageInfo{}, nil
	}
}

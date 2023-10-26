package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/blocks/
func GetBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetBlocks) ([]object.Account, error) {

	const location = "handler.mastodon_Blocks"

	return func(auth model.Authorization, t txn.GetBlocks) ([]object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return []object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Query the database
		userService := factory.User()
		users, err := userService.QueryBlockedUsers(auth.UserID, queryExpression(t))

		if err != nil {
			return []object.Account{}, derp.Wrap(err, location, "Error querying database")
		}

		// Convert the results to a slice of objects
		return sliceOfToots[model.User, object.Account](users), nil
	}
}

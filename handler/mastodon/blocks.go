package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/blocks/
func GetBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetBlocks) ([]object.Account, toot.PageInfo, error) {

	const location = "handler.mastodon.Rules"

	return func(auth model.Authorization, t txn.GetBlocks) ([]object.Account, toot.PageInfo, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return []object.Account{}, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return []object.Account{}, toot.PageInfo{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Query the database
		userService := factory.User()
		users, err := userService.QueryBlockedActors(session, auth.UserID, queryExpression(t))

		if err != nil {
			return []object.Account{}, toot.PageInfo{}, derp.Wrap(err, location, "Unable to query database")
		}

		// Convert the results to a slice of objects
		return getSliceOfToots(users), getPageInfo(users), nil
	}
}

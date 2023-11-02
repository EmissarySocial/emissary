package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/profile/
func DeleteProfile_Avatar(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Avatar) (object.Account, error) {

	const location = "handler.mastodon.DeleteProfile_Avatar"

	return func(auth model.Authorization, t txn.DeleteProfile_Avatar) (object.Account, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Error loading User", auth.UserID)
		}

		// Delete the user's Avatar
		if err := userService.DeleteAvatar(&user, "Deleted via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Error deleting Avatar")
		}

		return user.Toot(), nil
	}
}

func DeleteProfile_Header(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Header) (object.Account, error) {

	const location = "handler.mastodon.DeleteProfile_Header"

	return func(auth model.Authorization, t txn.DeleteProfile_Header) (object.Account, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Error loading User", auth.UserID)
		}

		// Nothing to do right now because Emissary doesn't track Header images.

		// Return their account as a Toot...
		return user.Toot(), nil
	}
}

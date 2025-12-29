package mastodon

import (
	"time"

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
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unable to load User", auth.UserID)
		}

		// Delete the user's Avatar
		if err := userService.DeleteAvatar(session, &user, "Deleted via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unable to delete Avatar")
		}

		return user.Toot(), nil
	}
}

func DeleteProfile_Header(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Header) (object.Account, error) {

	const location = "handler.mastodon.DeleteProfile_Header"

	return func(auth model.Authorization, t txn.DeleteProfile_Header) (object.Account, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unable to load User", auth.UserID)
		}

		// Nothing to do right now because Emissary doesn't track Header images.

		// Return their account as a Toot...
		return user.Toot(), nil
	}
}

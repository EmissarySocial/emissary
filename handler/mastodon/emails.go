package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/emails/
func PostEmailConfirmation(serverFactory *server.Factory) func(model.Authorization, txn.PostEmailConfirmation) (struct{}, error) {

	const location = "handler.mastodon.PostEmailConfirmation"

	return func(auth model.Authorization, t txn.PostEmailConfirmation) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error loading user")
		}

		// (Re-)send a welcome email to the User
		emailService := factory.Email()
		if err := emailService.SendPasswordReset(&user); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error sending welcome email")
		}

		// Success!
		return struct{}{}, nil
	}
}

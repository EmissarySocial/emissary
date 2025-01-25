package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
)

// StepSetPassword is a Step that can update a user's password
type StepSetPassword struct {
}

func (step StepSetPassword) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetPassword) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetPassword.Post"

	// Verify that the user is signed in.
	authorization := builder.authorization()

	if !authorization.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "You must be signed in to change your password"))
	}

	// Collect form POST information
	transaction, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form data"))
	}

	spew.Dump(transaction)

	// Load the User from the database
	factory := builder.factory()
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(authorization.UserID, &user); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error loading user"))
	}

	// Set the password (with Steranko password hasher)
	steranko := factory.Steranko()
	newPassword := transaction.Get("new_password")

	if err := steranko.SetPassword(&user, newPassword); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting password"))
	}
	spew.Dump(user)

	// Silence is AU-some
	return nil
}

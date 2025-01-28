package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
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

	// Collect form POST information
	transaction, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form data"))
	}

	// RULE: Verify that the user is trying to set a new password
	newPassword := transaction.Get("new_password")

	if newPassword == "" {
		return Continue()
	}

	// RULE: Users must be signed in, and can only change their own passwords.
	factory := builder.factory()
	steranko := factory.Steranko()
	authorization := builder.authorization()

	if !authorization.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "You must be signed in to change your password"))
	}

	// Load the User from the database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(authorization.UserID, &user); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error loading user"))
	}

	// Update the User's password using Steranko's default password hashing algorithm
	if err := steranko.SetPassword(&user, newPassword); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting password"))
	}

	// Save the User back to the database
	if err := userService.Save(&user, "Password changed"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving user"))
	}

	// Silence is AU-some
	return nil
}

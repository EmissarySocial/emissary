package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepSendEmail is a Step that can send a named email to a recipient
type StepSendEmail struct {
	Email string
}

func (step StepSendEmail) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post saves the object to the database
func (step StepSendEmail) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSendEmail.Post"

	// Confirm that we have a User builder object
	userBuilder, ok := builder.(User)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "Invalid Builder", "Builder must be Admin/User"))
	}

	// Send the designated email
	switch step.Email {

	case "welcome", "password-reset":
		builder.factory().User().SendPasswordResetEmail(builder.session(), userBuilder._user)

	default:
		return Halt().WithError(derp.InternalError(location, "Invalid email name", "Name must be 'welcome' or 'password-reset'"))
	}

	// Banana
	return nil
}

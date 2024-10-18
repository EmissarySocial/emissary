package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepSendEmail represents an action-step that can send a named email to a recipient
type StepSendEmail struct {
	Email string
}

func (step StepSendEmail) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post saves the object to the database
func (step StepSendEmail) Post(builder Builder, _ io.Writer) PipelineBehavior {

	// Confirm that we have a User builder object
	userBuilder, ok := builder.(User)

	if !ok {
		return Halt().WithError(derp.NewInternalError("build.StepSendEmail.Post", "Invalid Builder", "Builder must be Admin/User"))
	}

	// Collect required services
	factory := builder.factory()
	userService := factory.User()

	// Send the designated email
	switch step.Email {

	case "welcome", "password-reset":
		userService.SendPasswordResetEmail(userBuilder._user)

	default:
		return Halt().WithError(derp.NewInternalError("build.StepSendEmail.Post", "Invalid email name", "Name must be 'welcome' or 'password-reset'"))
	}

	// Banana
	return nil
}

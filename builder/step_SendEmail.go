package builder

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
	factory := builder.factory()
	emailService := factory.Email()

	userBuilder, ok := builder.(User)

	if !ok {
		return Halt().WithError(derp.NewInternalError("build.StepSendEmail.Post", "Invalid Builder", "Builder must be Admin/User"))
	}

	switch step.Email {

	case "welcome":

		if err := emailService.SendWelcome(userBuilder._user); err != nil {
			return Halt().WithError(err)
		}

	case "password-reset":
		if err := emailService.SendPasswordReset(userBuilder._user); err != nil {
			return Halt().WithError(err)
		}

	default:
		return Halt().WithError(derp.NewInternalError("build.StepSendEmail.Post", "Invalid email name", "Name must be 'welcome' or 'password-reset'"))
	}

	return nil
}

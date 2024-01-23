package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepSendEmail represents an action-step that can send a named email to a recipient
type StepSendEmail struct {
	Email string
}

func (step StepSendEmail) Get(_ Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post saves the object to the database
func (step StepSendEmail) Post(renderer Renderer, _ io.Writer) PipelineBehavior {
	factory := renderer.factory()
	emailService := factory.Email()

	userRenderer, ok := renderer.(User)

	if !ok {
		return Halt().WithError(derp.NewInternalError("render.StepSendEmail.Post", "Invalid Renderer", "Renderer must be Admin/User"))
	}

	switch step.Email {

	case "welcome":

		if err := emailService.SendWelcome(userRenderer._user); err != nil {
			return Halt().WithError(err)
		}

	case "password-reset":
		if err := emailService.SendPasswordReset(userRenderer._user); err != nil {
			return Halt().WithError(err)
		}

	default:
		return Halt().WithError(derp.NewInternalError("render.StepSendEmail.Post", "Invalid email name", "Name must be 'welcome' or 'password-reset'"))
	}

	return nil
}

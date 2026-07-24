package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
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
		return Halt().WithError(derp.Internal(location, "Invalid Builder", "Builder must be Admin/User"))
	}

	// Send the designated email
	switch step.Email {

	case "welcome":
		// Report-and-continue: a bounced welcome email must not fail the action that triggered it.
		if err := builder.factory().User().SendPasswordResetEmail(builder.session(), userBuilder._user, model.PasswordResetDurationWelcome); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending welcome email", userBuilder._user.Username))
		}

	case "password-reset":
		// Report-and-continue: this step runs from admin/self flows where the reset code is still
		// issued; the send failure is logged rather than blocking the surrounding action.
		if err := builder.factory().User().SendPasswordResetEmail(builder.session(), userBuilder._user, model.PasswordResetDurationReset); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending password reset email", userBuilder._user.Username))
		}

	default:
		return Halt().WithError(derp.Internal(location, "Invalid email name", "Name must be 'welcome' or 'password-reset'"))
	}

	// Banana
	return nil
}

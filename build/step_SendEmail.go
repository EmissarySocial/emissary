package build

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// StepSendEmail is a Step that can send a named email to a recipient
type StepSendEmail struct {
	Email  string
	Values map[string]*template.Template
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSendEmail) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post sends the email named by this step
func (step StepSendEmail) Post(builder Builder, _ io.Writer) PipelineBehavior {

	// The built-in emails each mint a credential before they send, so they route
	// through the User service instead of the generic path
	if modelStep.IsBuiltInEmail(step.Email) {
		return step.postBuiltIn(builder)
	}

	return step.postTemplateEmail(builder)
}

// postTemplateEmail sends any email defined by a Template, with values rendered by this step
func (step StepSendEmail) postTemplateEmail(builder Builder) PipelineBehavior {

	const location = "build.StepSendEmail.postTemplateEmail"

	// Render the values that this step passes into the email
	values := make(mapof.Any, len(step.Values))

	for key, valueTemplate := range step.Values {
		values[key] = executeTemplate(valueTemplate, builder)
	}

	// RULE: a send failure halts the pipeline.  Nothing here is stored or queued, so an error
	// reported-and-continued is a message that vanishes with nobody aware it ever existed.
	if err := builder.factory().Email().Send(step.Email, values); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error sending email", step.Email))
	}

	// Return to sender
	return nil
}

// postBuiltIn sends one of the two emails that require a freshly minted credential
func (step StepSendEmail) postBuiltIn(builder Builder) PipelineBehavior {

	const location = "build.StepSendEmail.postBuiltIn"

	// Confirm that we have a User builder object
	userBuilder, ok := builder.(User)

	if !ok {
		return Halt().WithError(derp.Internal(location, "Invalid Builder", "Builder must be Admin/User"))
	}

	// Choose the reset window that matches the email being sent
	duration := model.PasswordResetDurationReset

	if step.Email == "welcome" {
		duration = model.PasswordResetDurationWelcome
	}

	// Report-and-continue: these run from admin/self flows where the reset code is still issued,
	// so a bounced email is logged rather than failing the surrounding action.  This is the
	// opposite of postTemplateEmail, where nothing else records that the message existed.
	if err := builder.factory().User().SendPasswordResetEmail(builder.session(), userBuilder._user, duration); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending email", step.Email, userBuilder._user.Username))
	}

	// Banana
	return nil
}

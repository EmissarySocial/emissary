package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithImport is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithImport struct {
	SubSteps []step.Step
}

func (step StepWithImport) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithImport) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithImport) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithImport.execute"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	record, err := step.getImport(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to get Import record"))
	}

	// Create a new builder tied to the Import record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &record, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)

	if result.Error != nil {
		return Halt().WithError(derp.Wrap(result.Error, location, "Error executing steps for child"))
	}

	return UseResult(result)
}

func (step StepWithImport) getImport(builder Builder) (model.Import, error) {

	const location = "build.StepWithImport.getImport"

	// Locate the User ID for this request
	userID := builder.AuthenticatedID()

	// Collect required services and values
	importService := builder.factory().Import()
	record := model.NewImport()
	record.UserID = userID

	token := builder.QueryParam("importId")

	// If token is empty, then create a new `Import` record
	if isNewOrEmpty(token) {
		return record, nil
	}

	// Otherwise, try to load the Import record from the database
	if err := importService.LoadByToken(builder.session(), userID, token, &record); err != nil {
		return record, derp.Wrap(err, location, "Unable to load Import", token)
	}

	// Success.
	return record, nil
}

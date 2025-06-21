package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithAttachment is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithAttachment struct {
	SubSteps []step.Step
}

func (step StepWithAttachment) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithAttachment) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithAttachment) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithAttachment.execute"

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Get Object info from the parent builder
	objectID := builder.objectID()
	objectType := builder.objectType()

	// Collect required services and values
	factory := builder.factory()
	attachmentService := factory.Attachment()
	attachment := model.NewAttachment(objectType, objectID)
	token := builder.QueryParam("attachmentId")

	// Only authenticated users can create new Attachment records
	if (token == "") || (token == "new") {

		if !builder.IsAuthenticated() {
			return Halt().WithError(derp.ForbiddenError(location, "Anonymous user is not authorized to perform this action"))
		}

	} else if attachmentID, err := primitive.ObjectIDFromHex(token); err == nil {

		if err := attachmentService.LoadByID(objectType, objectID, attachmentID, &attachment); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Unable to load Attachment via ID", token))
		}
	}

	// Create a new builder tied to the Attachment record
	subBuilder, err := NewAttachment(factory, builder.request(), builder.response(), template, &attachment, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the build pipeline on the Attachment record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

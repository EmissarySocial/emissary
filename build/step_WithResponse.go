package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithResponse is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithResponse struct {
	SubSteps []step.Step
}

func (step StepWithResponse) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithResponse) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithResponse) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithResponse.doStep"

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
	responseService := factory.Response()
	responseToken := builder.QueryParam("responseId")
	response := model.NewResponse()

	// If we have a real ID, then try to load the response from the database
	if (responseToken != "") && (responseToken != "new") {
		if responseID, err := primitive.ObjectIDFromHex(responseToken); err == nil {
			if err := responseService.LoadByID(builder.session(), builder.AuthenticatedID(), responseID, &response); err != nil {
				if actionMethod == ActionMethodGet {
					return Halt().WithError(derp.Wrap(err, location, "Unable to load Response", responseToken))
				}
				// Fall through for POSTS..  we're just creating a new response.
			}
		}
	}

	// Create a new builder tied to the Response record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &response, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

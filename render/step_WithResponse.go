package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithResponse represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithResponse struct {
	SubSteps []step.Step
}

func (step StepWithResponse) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithResponse) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithResponse) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithResponse.doStep"

	if !renderer.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := renderer.factory()
	context := renderer.context()
	responseService := factory.Response()
	responseToken := context.QueryParam("responseId")
	response := model.NewResponse()

	// If we have a real ID, then try to load the response from the database
	if (responseToken != "") && (responseToken != "new") {
		if responseID, err := primitive.ObjectIDFromHex(responseToken); err == nil {
			if err := responseService.LoadByID(responseID, &response); err != nil {
				if actionMethod == ActionMethodGet {
					return Halt().WithError(derp.Wrap(err, location, "Unable to load Response", responseToken))
				}
				// Fall through for POSTS..  we're just creating a new response.
			}
		}
	}

	// Create a new renderer tied to the Response record
	subRenderer, err := NewModel(factory, context, responseService, &response, renderer.template(), renderer.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

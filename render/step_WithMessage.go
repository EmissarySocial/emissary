package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithMessage is an action-step that executes a new pipeline on an Inbox Message, identified by the query parameter "messageId"
type StepWithMessage struct {
	SubSteps []step.Step
}

func (step StepWithMessage) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the message with data from the request body.
func (step StepWithMessage) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithMessage) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithMessage.execute"

	if !renderer.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Parse the MessageID from the query string
	messageID, err := primitive.ObjectIDFromHex(renderer.QueryParam("messageId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "MessageID must be a valid hex string"))
	}

	// Collect required services and values
	factory := renderer.factory()
	inboxService := factory.Inbox()
	message := model.NewMessage()
	userID := renderer.AuthenticatedID()

	// If we have a real ID, then try to load the message from the database
	if err := inboxService.LoadByID(userID, messageID, &message); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load Message", messageID))
	}

	// Create a new renderer tied to the Message record
	subRenderer, err := NewModel(factory, renderer.request(), renderer.response(), inboxService, &message, renderer.template(), renderer.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithMessage represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithMessage struct {
	SubSteps []step.Step
}

func (step StepWithMessage) Get(renderer Renderer, buffer io.Writer) error {
	return step.doStep(renderer, buffer, ActionMethodGet)
}

func (step StepWithMessage) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post updates the stream with approved data from the request body.
func (step StepWithMessage) Post(renderer Renderer) error {
	return step.doStep(renderer, nil, ActionMethodPost)
}

func (step StepWithMessage) doStep(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "render.StepWithMessage.doStep"

	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action")
	}

	// Collect required services and values
	factory := renderer.factory()
	context := renderer.context()
	inboxService := factory.Inbox()
	messageToken := context.QueryParam("messageId")
	message := model.NewMessage()
	message.UserID = renderer.AuthenticatedID()

	// If we have a real ID, then try to load the message from the database
	if messageToken != "new" {

		messageID, err := primitive.ObjectIDFromHex(messageToken)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Message ID", messageToken)
		}

		if err := inboxService.LoadByID(renderer.AuthenticatedID(), messageID, &message); err != nil {
			return derp.Wrap(err, location, "Unable to load Message", messageToken)
		}
	}

	// Create a new renderer tied to the Message record
	subRenderer, err := NewModel(factory, context, inboxService, &message, renderer.template(), renderer.ActionID())

	if err != nil {
		return derp.Wrap(err, location, "Unable to create sub-renderer")
	}

	// Execute the POST render pipeline on the child
	if err := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error executing steps for child")
	}

	return nil
}

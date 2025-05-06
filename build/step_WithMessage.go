package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithMessage is a Step that executes a new pipeline on an Inbox Message, identified by the query parameter "messageId"
type StepWithMessage struct {
	SubSteps []step.Step
}

func (step StepWithMessage) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the message with data from the request body.
func (step StepWithMessage) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithMessage) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithMessage.execute"

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

	// Parse the MessageID from the query string
	messageID, err := primitive.ObjectIDFromHex(builder.QueryParam("messageId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "MessageID must be a valid hex string"))
	}

	// Collect required services and values
	factory := builder.factory()
	inboxService := factory.Inbox()
	message := model.NewMessage()
	userID := builder.AuthenticatedID()

	// If we have a real ID, then try to load the message from the database
	if err := inboxService.LoadByID(userID, messageID, &message); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load Message", messageID))
	}

	// Create a new builder tied to the Message record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &message, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

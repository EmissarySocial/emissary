package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithConversation is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithConversation struct {
	SubSteps []step.Step
}

func (step StepWithConversation) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithConversation) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithConversation) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithConversation.doStep"

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
	conversation := model.NewConversation()
	conversation.UserID = builder.AuthenticatedID()

	if token := builder.QueryParam("conversationId"); isNewOrEmpty(token) {

		conversationID, err := primitive.ObjectIDFromHex(token)
		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid Conversation ID", token))
		}

		if err := factory.Conversation().LoadByID(builder.session(), builder.AuthenticatedID(), conversationID, &conversation); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Conversation", conversationID))
			}
			// Fall through for POSTS..  we're just creating a new conversation.
		}
	}

	// Create a new builder tied to the Conversation record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &conversation, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

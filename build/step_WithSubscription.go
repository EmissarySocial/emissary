package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithSubscription is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithSubscription struct {
	SubSteps []step.Step
}

func (step StepWithSubscription) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithSubscription) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithSubscription) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithSubscription.doStep"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.NewInternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	subscriptionService := factory.Subscription()
	subscriptionToken := builder.QueryParam("subscriptionId")
	subscription := model.NewSubscription()
	subscription.UserID = builder.AuthenticatedID()

	if (subscriptionToken != "") && (subscriptionToken != "new") {
		if err := subscriptionService.LoadByUserAndToken(builder.AuthenticatedID(), subscriptionToken, &subscription); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Subscription", subscriptionToken))
			}
			// Fall through for POSTS..  we're just creating a new subscription.
		}
	}

	// Create a new builder tied to the Subscription record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &subscription, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	reesult := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	reesult.Error = derp.Wrap(reesult.Error, location, "Error executing steps for child")

	return UseResult(reesult)
}

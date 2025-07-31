package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithMerchantAccount is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithMerchantAccount struct {
	SubSteps []step.Step
}

func (step StepWithMerchantAccount) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithMerchantAccount) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithMerchantAccount) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithMerchantAccount.doStep"

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
	merchantAccountService := factory.MerchantAccount()
	merchantAccountToken := builder.QueryParam("merchantAccountId")
	merchantAccount := model.NewMerchantAccount()
	merchantAccount.UserID = builder.AuthenticatedID()

	if (merchantAccountToken != "") && (merchantAccountToken != "new") {
		if err := merchantAccountService.LoadByUserAndToken(builder.session(), builder.AuthenticatedID(), merchantAccountToken, &merchantAccount); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load MerchantAccount", merchantAccountToken))
			}
			// Fall through for POSTS..  we're just creating a new merchantAccount.
		}
	}

	// Create a new builder tied to the MerchantAccount record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &merchantAccount, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

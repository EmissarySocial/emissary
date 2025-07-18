package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithRule is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithRule struct {
	SubSteps []step.Step
}

func (step StepWithRule) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithRule) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithRule) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithRule.doStep"

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
	ruleService := factory.Rule()
	ruleToken := builder.QueryParam("ruleId")
	rule := model.NewRule()
	rule.UserID = builder.AuthenticatedID()

	if (ruleToken != "") && (ruleToken != "new") {
		if err := ruleService.LoadByToken(builder.AuthenticatedID(), ruleToken, &rule); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Rule", ruleToken))
			}
			// Fall through for POSTS..  we're just creating a new rule.
		}
	}

	// Create a new builder tied to the Rule record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &rule, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

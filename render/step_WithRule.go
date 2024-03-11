package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithRule represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithRule struct {
	SubSteps []step.Step
}

func (step StepWithRule) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithRule) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithRule) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithRule.doStep"

	if !renderer.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := renderer.factory()
	ruleService := factory.Rule()
	ruleToken := renderer.QueryParam("ruleId")
	rule := model.NewRule()
	rule.UserID = renderer.AuthenticatedID()

	if (ruleToken != "") && (ruleToken != "new") {
		if err := ruleService.LoadByToken(renderer.AuthenticatedID(), ruleToken, &rule); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Rule", ruleToken))
			}
			// Fall through for POSTS..  we're just creating a new rule.
		}
	}

	// Create a new renderer tied to the Rule record
	subRenderer, err := NewModel(factory, renderer.request(), renderer.response(), &rule, renderer.template(), renderer.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	reesult := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	reesult.Error = derp.Wrap(reesult.Error, location, "Error executing steps for child")

	return UseResult(reesult)
}

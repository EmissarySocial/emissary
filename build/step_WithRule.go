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

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithRule) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithRule) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithRule) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithRule.doStep"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()

	rule, err := step.getRule(builder)
	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading Rule"))
	}

	// Create a new builder tied to the Rule record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &rule, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for child")

	return UseResult(result)
}

// getRule loads the Rule named in the query string, or returns a new one
func (step StepWithRule) getRule(builder Builder) (model.Rule, error) {

	rule := model.NewRule()
	rule.UserID = builder.AuthenticatedID()
	rule.Action = ""

	if token := builder.QueryParam("ruleId"); notNewOrEmpty(token) {
		if err := builder.factory().Rule().LoadByToken(builder.session(), builder.AuthenticatedID(), token, &rule); err != nil {
			if !derp.IsNotFound(err) {
				return rule, derp.Wrap(err, "build.StepWithRule.getRule", "Loading Rule with token "+token)
			}
		}
		return rule, nil
	}

	if token := builder.QueryParam("actor"); token != "" {

		if err := builder.factory().Rule().LoadByMatchKey(builder.session(), builder.AuthenticatedID(), model.RuleTypeActor, token, &rule); err != nil {
			if !derp.IsNotFound(err) {
				return rule, derp.Wrap(err, "build.StepWithRule.getRule", "Loading Rule with actor "+token)
			}
		}

		// Seed the Trigger for a brand-new Rule so Save can resolve its (Required) MatchKey. An
		// existing Rule already carries the friendly Trigger the User originally typed, so leave it.
		if rule.IsNew() {
			rule.Trigger = token
		}

		return rule, nil
	}

	return rule, nil
}

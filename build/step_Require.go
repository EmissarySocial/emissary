package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepRequire is a Step that halts the pipeline with a 400 Bad Request error unless its condition is true
type StepRequire struct {
	Condition *template.Template
}

// Get halts a GET request unless the condition is true. Implements the Step interface.
func (step StepRequire) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return step.execute(builder)
}

// Post halts a POST request unless the condition is true. Implements the Step interface.
func (step StepRequire) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return step.execute(builder)
}

// execute evaluates the condition, and halts with an error unless it is true
func (step StepRequire) execute(builder Builder) PipelineBehavior {

	const location = "build.StepRequire.execute"

	// Evaluate the condition, which the "if" step reads the same way
	condition, err := evaluateCondition(step.Condition, builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Evaluating condition"))
	}

	// RULE: Anything but a "true" condition refuses the request
	if !condition {
		return Halt().WithError(derp.BadRequest(location, "Requirement not met"))
	}

	// You shall pass
	return nil
}

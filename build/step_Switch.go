package build

import (
	"bytes"
	"io"
	"strings"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepSwitch is a Step that renders an expression, then runs the sub-pipeline of the case whose
// key matches it, or the default sub-pipeline when no case matches
type StepSwitch struct {
	Expression *template.Template
	Cases      map[string][]step.Step
	Default    []step.Step
}

// Get runs the matching sub-pipeline during a GET request. Implements the Step interface.
func (step StepSwitch) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post runs the matching sub-pipeline during a POST request. Implements the Step interface.
func (step StepSwitch) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute renders the expression and runs the sub-pipeline it selects, in the current phase
func (step StepSwitch) execute(builder Builder, buffer io.Writer, method ActionMethod) PipelineBehavior {

	const location = "build.StepSwitch.execute"

	// Render the expression
	value, err := step.evaluateExpression(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Evaluating expression"))
	}

	// Run the matching case.  The error names the case key, never the rendered value.
	if steps, exists := step.Cases[value]; exists {
		result := Pipeline(steps).Execute(builder.factory(), builder, buffer, method)
		result.Error = derp.WrapIF(result.Error, location, "Executing case", value)
		return UseResult(result)
	}

	// Otherwise, run the default
	result := Pipeline(step.Default).Execute(builder.factory(), builder, buffer, method)
	result.Error = derp.WrapIF(result.Error, location, "Executing 'default' sub-steps")
	return UseResult(result)
}

// evaluateExpression renders the expression against the builder, without surrounding whitespace
func (step StepSwitch) evaluateExpression(builder Builder) (string, error) {

	var buffer bytes.Buffer

	if err := step.Expression.Execute(&buffer, builder); err != nil {
		return "", derp.Wrap(err, "build.StepSwitch.evaluateExpression", "Executing template")
	}

	return strings.TrimSpace(buffer.String()), nil
}

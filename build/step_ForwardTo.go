package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepForwardTo is a Step that sends a visitor to another URL, because they are finished
// with this page. See [navigation.go] for how that reaches the client.
type StepForwardTo struct {
	URL    *template.Template
	Method string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepForwardTo) Get(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "get") || (step.Method == "both") {
		return step.execute(builder)
	}

	return Continue()
}

// Post renders this step during a POST request. Implements the Step interface.
func (step StepForwardTo) Post(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "post") || (step.Method == "both") {
		return step.execute(builder)
	}

	return Continue()
}

// execute sends the visitor to this Step's target URL.
func (step StepForwardTo) execute(builder Builder) PipelineBehavior {

	const location = "build.StepForwardTo.execute"

	target, err := navigationURL(step.URL, builder, location)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Building forward target"))
	}

	return navigateDocument(builder, target)
}

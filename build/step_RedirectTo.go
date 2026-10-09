package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepRedirectTo is a Step that sends a visitor to another URL, because the content they
// asked for lives there. See [navigation.go] for how that reaches the client.
type StepRedirectTo struct {
	StatusCode int
	URL        *template.Template
	Method     string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepRedirectTo) Get(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "get") || (step.Method == "both") {
		return step.execute(builder)
	}

	return Continue()
}

// Post renders this step during a POST request. Implements the Step interface.
func (step StepRedirectTo) Post(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "post") || (step.Method == "both") {
		return step.execute(builder)
	}

	return Continue()
}

// execute sends the visitor to this Step's target URL.
func (step StepRedirectTo) execute(builder Builder) PipelineBehavior {

	const location = "build.StepRedirectTo.execute"

	target, err := navigationURL(step.URL, builder, location)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Building redirect target"))
	}

	return navigateContent(builder, step.StatusCode, target)
}

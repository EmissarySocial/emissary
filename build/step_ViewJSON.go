package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepViewJSON is a Step that can build a Stream into HTML
type StepViewJSON struct {
	Value *template.Template
}

// Get builds the Stream HTML to the context
func (step StepViewJSON) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if err := step.Value.Execute(buffer, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepViewJSON.Get", "Unable to execute template"))
	}

	result := Continue()
	result = result.AsFullPage()

	// Otherwise, just continue without headers.
	return result
}

func (step StepViewJSON) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return Continue()
}

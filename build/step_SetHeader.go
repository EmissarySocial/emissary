package build

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepSetHeader is a Step that can update the custom data stored in a Stream
type StepSetHeader struct {
	Method     string
	HeaderName string
	Value      *template.Template
}

func (step StepSetHeader) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	if step.Method == "post" {
		return nil
	}
	return step.setHeader(builder)
}

// Post updates the stream with approved data from the request body.
func (step StepSetHeader) Post(builder Builder, _ io.Writer) PipelineBehavior {
	if step.Method == "get" {
		return nil
	}
	return step.setHeader(builder)
}

func (step StepSetHeader) setHeader(builder Builder) PipelineBehavior {

	var value bytes.Buffer

	if err := step.Value.Execute(&value, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetHeader.Post", "Error executing template", step.Value))
	}

	builder.response().Header().Set(step.HeaderName, value.String())

	return nil
}

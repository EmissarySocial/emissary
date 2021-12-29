package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// StepSetData represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetData struct {
	paths  []string
	values datatype.Map
}

func NewStepSetData(stepInfo datatype.Map) StepSetData {

	return StepSetData{
		paths:  stepInfo.GetSliceOfString("paths"),
		values: stepInfo.GetMap("values"),
	}
}

// Get does not display anything.
func (step StepSetData) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(buffer io.Writer, renderer Renderer) error {

	// Try to find the schema for this Template
	schema := renderer.schema()
	inputs := make(datatype.Map)

	// Collect form POST information
	if err := renderer.context().Bind(&inputs); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepSetData.Post", "Error binding body")
	}

	if err := schema.Validate(inputs); err != nil {
		return derp.Wrap(err, "ghost.render.StepSetData.Post", "Error validating input", inputs)
	}

	// Put approved form data into the stream
	for _, p := range step.paths {
		if err := renderer.SetPath(path.New(p), inputs[p]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepSetData.Post", "Error seting value from user input", p)
		}
	}

	// Put values from schema.json into the stream
	if err := path.SetAll(renderer, step.values); err != nil {
		return derp.Wrap(err, "ghost.render.StepSetData.Post", "Error setting value from schema.json", step.values)
	}

	return nil
}

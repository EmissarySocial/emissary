package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
)

// StepSetData represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetData struct {
	templateService *service.Template
	streamService   *service.Stream
	paths           []string
	values          datatype.Map
}

func NewStepSetData(templateService *service.Template, streamService *service.Stream, formLibrary form.Library, stepInfo datatype.Map) StepSetData {

	return StepSetData{
		templateService: templateService,
		streamService:   streamService,
		paths:           stepInfo.GetSliceOfString("paths"),
		values:          stepInfo.GetMap("values"),
	}
}

// Get does not display anything.
func (step StepSetData) Get(buffer io.Writer, renderer *Stream) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(buffer io.Writer, renderer *Stream) error {

	// Try to find the schema for this Template
	schema, err := step.templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepForm.Get", "Invalid Schema")
	}

	inputs := make(datatype.Map)

	// Collect form POST information
	if err := renderer.ctx.Bind(&inputs); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepForm.Post", "Error binding body")
	}

	if err := schema.Validate(inputs); err != nil {
		return derp.Wrap(err, "ghost.render.StepForm.Post", "Error validating input", inputs)
	}

	// Put approved form data into the stream
	for _, p := range step.paths {
		if err := path.Set(renderer.stream, p, inputs[p]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepSetData.Post", "Error seting value from user input", p)
		}
	}

	// Put values from schema.json into the stream
	for key, value := range step.values {
		if err := path.Set(renderer.stream, key, value); err != nil {
			return derp.Wrap(err, "ghose.render.StepSetData.Post", "Error setting value from schema.json", key, value)
		}
	}

	return nil
}

package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
)

// StepStreamData represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepStreamData struct {
	templateService *service.Template
	streamService   *service.Stream
	paths           []string
}

func NewStepStreamData(templateService *service.Template, streamService *service.Stream, formLibrary form.Library, stepInfo datatype.Map) StepStreamData {

	return StepStreamData{
		templateService: templateService,
		streamService:   streamService,
		paths:           stepInfo.GetSliceOfString("paths"),
	}
}

// Get displays a form where users can update stream data
func (step StepStreamData) Get(buffer io.Writer, renderer *Stream) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStreamData) Post(buffer io.Writer, renderer *Stream) error {

	// Put approved form data into the stream
	for _, p := range step.paths {
		if err := path.Set(renderer.stream, p, renderer.inputs[p]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamData.Post", "Error seting value", p)
		}
	}

	return nil
}

package render

import (
	"io"
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/labstack/echo/v4"
)

// StepSetData represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetData struct {
	paths    []string     // List of paths to pull from form data
	values   datatype.Map // values to set directly into the object
	defaults datatype.Map // values to set into the object IFF they are currently empty.
}

func NewStepSetData(stepInfo datatype.Map) StepSetData {

	return StepSetData{
		paths:    stepInfo.GetSliceOfString("paths"),
		values:   stepInfo.GetMap("values"),
		defaults: stepInfo.GetMap("defaults"),
	}
}

// Get does not display anything.
func (step StepSetData) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(buffer io.Writer, renderer Renderer) error {

	object := renderer.object()

	// Try to find the schema for this Template
	schema := renderer.schema()
	inputs := make(datatype.Map)

	// Collect form POST information
	if err := (&echo.DefaultBinder{}).BindBody(renderer.context(), &inputs); err != nil {
		result := derp.Wrap(err, "render.StepSetData.Post", "Error binding body")
		derp.SetErrorCode(result, http.StatusBadRequest)
		return result
	}

	// Put approved form data into the stream
	for _, p := range step.paths {
		if err := schema.Set(object, p, inputs[p]); err != nil {
			result := derp.Wrap(err, "render.StepSetData.Post", "Error seting value from user input", inputs, p)
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}
	}

	// Put values from schema.json into the stream
	for key, value := range step.values {
		if err := schema.Set(object, key, value); err != nil {
			result := derp.Wrap(err, "render.StepSetData.Post", "Error setting value from schema.json", step.values)
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.defaults {
		if convert.IsZeroValue(path.Get(renderer, name)) {
			if err := schema.Set(object, name, value); err != nil {
				result := derp.Wrap(err, "render.StepSetData.Post", "Error setting default value", name, value)
				derp.SetErrorCode(result, http.StatusBadRequest)
				return result
			}
		}
	}

	// Silence is Golden
	return nil
}

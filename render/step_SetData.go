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

// StepSetData represents an action-step that can update the custom data stored in a Stream
type StepSetData struct {
	Paths    []string     // List of paths to pull from form data
	Values   datatype.Map // values to set directly into the object
	Defaults datatype.Map // values to set into the object IFF they are currently empty.
}

func (step StepSetData) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepSetData) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(renderer Renderer) error {

	const location = "render.StepSetData.Post"

	object := renderer.object()

	// Try to find the schema for this Template
	schema := renderer.schema()
	inputs := make(datatype.Map)

	// Collect form POST information
	if err := (&echo.DefaultBinder{}).BindBody(renderer.context(), &inputs); err != nil {
		result := derp.Wrap(err, location, "Error binding body")
		derp.SetErrorCode(result, http.StatusBadRequest)
		return result
	}

	// Put approved form data into the stream
	for _, p := range step.Paths {
		if err := schema.Set(object, p, inputs[p]); err != nil {
			result := derp.Wrap(err, location, "Error seting value from user input", inputs, p)
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}
	}

	// Put values from schema.json into the stream
	for key, value := range step.Values {
		if err := schema.Set(object, key, value); err != nil {
			result := derp.Wrap(err, location, "Error setting value from schema.json", key, value)
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.Defaults {
		if convert.IsZeroValue(path.Get(renderer, name)) {
			if err := schema.Set(object, name, value); err != nil {
				result := derp.Wrap(err, location, "Error setting default value", name, value)
				derp.SetErrorCode(result, http.StatusBadRequest)
				return result
			}
		}
	}

	// Silence is AU-some
	return nil
}

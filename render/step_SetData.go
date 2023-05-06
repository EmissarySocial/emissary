package render

import (
	"io"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

// StepSetData represents an action-step that can update the custom data stored in a Stream
type StepSetData struct {
	FromURL  []string  // List of paths to pull from URL data
	FromForm []string  // List of paths to pull from Form data
	Values   mapof.Any // values to set directly into the object
	Defaults mapof.Any // values to set into the object IFF they are currently empty.
}

func (step StepSetData) Get(renderer Renderer, buffer io.Writer) error {

	if err := step.setURLPaths(renderer); err != nil {
		return derp.Wrap(err, "render.StepSetData.Get", "Error setting data from URL")
	}

	return nil
}

func (step StepSetData) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepSetData.Post"

	if err := step.setURLPaths(renderer); err != nil {
		return derp.Wrap(err, "render.StepSetData.Get", "Error setting data from URL")
	}

	object := renderer.object()
	schema := renderer.schema()

	if len(step.FromForm) > 0 {

		inputs := mapof.NewAny()

		// Collect form POST information
		if err := (&echo.DefaultBinder{}).BindBody(renderer.context(), &inputs); err != nil {
			result := derp.Wrap(err, location, "Error binding body")
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}

		// Put approved form data into the stream
		for _, p := range step.FromForm {
			if err := schema.Set(object, p, inputs[p]); err != nil {
				result := derp.Wrap(err, location, "Error seting value from user input", inputs, p)
				derp.SetErrorCode(result, http.StatusBadRequest)
				return result
			}
		}
	}

	// Put values from template.json into the stream
	for key, value := range step.Values {
		if err := schema.Set(object, key, value); err != nil {
			result := derp.Wrap(err, location, "Error setting value from template.json", key, value)
			derp.SetErrorCode(result, http.StatusBadRequest)
			return result
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.Defaults {
		currentValue, _ := schema.Get(renderer, name)
		if convert.IsZeroValue(currentValue) {
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

func (step StepSetData) setURLPaths(renderer Renderer) error {

	if len(step.FromURL) > 0 {
		query := renderer.context().Request().URL.Query()
		schema := renderer.schema()
		object := renderer.object()
		for _, path := range step.FromURL {
			if value := query.Get(path); value != "" {
				if err := schema.Set(object, path, value); err != nil {
					result := derp.Wrap(err, "render.StepSetData.setURLPaths", "Error setting data from URL")
					derp.SetErrorCode(result, http.StatusBadRequest)
					return result
				}
			}
		}
	}

	return nil
}

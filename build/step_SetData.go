package build

import (
	"io"
	"net/http"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/mapof"
)

// StepSetData represents an action-step that can update the custom data stored in a Stream
type StepSetData struct {
	FromURL  []string                      // List of paths to pull from URL data
	FromForm []string                      // List of paths to pull from Form data
	Values   map[string]*template.Template // values to set directly into the object
	Defaults mapof.Any                     // values to set into the object IFF they are currently empty.
}

func (step StepSetData) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if err := step.setURLPaths(builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetData.Get", "Error setting data from URL"))
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetData.Post"

	if err := step.setURLPaths(builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetData.Get", "Error setting data from URL"))
	}

	object := builder.object()
	schema := builder.schema()

	if len(step.FromForm) > 0 {

		transaction := mapof.NewAny()

		// Collect form POST information
		if err := bindBody(builder.request(), &transaction); err != nil {
			result := derp.Wrap(err, location, "Error binding body", derp.WithCode(http.StatusBadRequest))
			return Halt().WithError(result)
		}

		// Put approved form data into the stream
		for _, p := range step.FromForm {
			if err := schema.Set(object, p, transaction[p]); err != nil {
				result := derp.Wrap(err, location, "Error seting value from user input", transaction, p, derp.WithCode(http.StatusBadRequest))
				return Halt().WithError(result)
			}
		}
	}

	// Put values from template.json into the stream
	for key, value := range step.Values {
		valueString := executeTemplate(value, builder)
		if err := schema.Set(object, key, valueString); err != nil {
			result := derp.Wrap(err, location, "Error setting value from template.json", key, derp.WithCode(http.StatusBadRequest))
			return Halt().WithError(result)
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.Defaults {
		currentValue, _ := schema.Get(builder, name)
		if compare.IsZero(currentValue) {
			if err := schema.Set(object, name, value); err != nil {
				result := derp.Wrap(err, location, "Error setting default value", name, value, derp.WithCode(http.StatusBadRequest))
				return Halt().WithError(result)
			}
		}
	}

	// Silence is AU-some
	return nil
}

func (step StepSetData) setURLPaths(builder Builder) error {

	if len(step.FromURL) > 0 {
		query := builder.request().URL.Query()
		schema := builder.schema()
		object := builder.object()
		for _, path := range step.FromURL {
			if value := query.Get(path); value != "" {
				if err := schema.Set(object, path, value); err != nil {
					result := derp.Wrap(err, "build.StepSetData.setURLPaths", "Error setting data from URL", derp.WithCode(http.StatusBadRequest))
					return result
				}
			}
		}
	}

	return nil
}

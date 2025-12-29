package build

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/mapof"
)

// StepSetData is a Step that can update the custom data stored in a Stream
type StepSetData struct {
	FromURL  []string                      // List of paths to pull from URL data
	FromForm []string                      // List of paths to pull from Form data
	Values   map[string]*template.Template // values to set directly into the object
	Defaults mapof.Any                     // values to set into the object IFF they are currently empty.
}

func (step StepSetData) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetData.Get"

	if err := step.setURLPaths(builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetData.Get", "Unable to set data from URL"))
	}

	object := builder.object()
	schema := builder.schema()

	// Put values from template.json into the stream
	for key, value := range step.Values {
		valueString := executeTemplate(value, builder)
		if err := schema.Set(object, key, valueString); err != nil {
			result := derp.Wrap(err, location, "Unable to set value from template.json", key, derp.WithBadRequest())
			return Halt().WithError(result)
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.Defaults {
		if currentValue, _ := schema.Get(builder, name); compare.IsZero(currentValue) {
			if err := schema.Set(object, name, value); err != nil {
				result := derp.Wrap(err, location, "Unable to set default value", name, value, derp.WithBadRequest())
				return Halt().WithError(result)
			}
		}
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetData) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetData.Post"

	if err := step.setURLPaths(builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetData.Get", "Unable to set data from URL"))
	}

	object := builder.object()
	schema := builder.schema()

	if len(step.FromForm) > 0 {

		// Collect form POST information
		transaction, err := formdata.Parse(builder.request())

		if err != nil {
			result := derp.Wrap(err, location, "Error binding body", derp.WithBadRequest())
			return Halt().WithError(result)
		}

		// Put approved form data into the stream
		for _, p := range step.FromForm {
			if err := schema.Set(object, p, transaction[p]); err != nil {
				result := derp.Wrap(err, location, "Error seting value from user input", transaction, p, derp.WithBadRequest())
				return Halt().WithError(result)
			}
		}
	}

	// Put values from template.json into the stream
	for key, value := range step.Values {
		valueString := executeTemplate(value, builder)
		if err := schema.Set(object, key, valueString); err != nil {
			result := derp.Wrap(err, location, "Unable to set value from template.json", key, derp.WithBadRequest())
			return Halt().WithError(result)
		}
	}

	// Set default values (only if no value already exists)
	for name, value := range step.Defaults {
		if currentValue, _ := schema.Get(builder, name); compare.IsZero(currentValue) {
			if err := schema.Set(object, name, value); err != nil {
				result := derp.Wrap(err, location, "Unable to set default value", name, value, derp.WithBadRequest())
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
					result := derp.Wrap(err, "build.StepSetData.setURLPaths", "Unable to set data from URL", derp.WithBadRequest())
					return result
				}
			}
		}
	}

	return nil
}

package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepAddModelObject is an action that can add new model objects of any type
type StepAddModelObject struct {
	Form     form.Element
	Defaults []step.Step
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepAddModelObject) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	factory := builder.factory()
	schema := builder.schema()
	object := builder.object()

	// First, try to execute any "default" steps so that the object is initialized
	result := Pipeline(step.Defaults).Get(factory, builder, buffer)

	if result.Halt {
		result.Error = derp.Wrap(result.Error, "build.StepAddModelObject.Get", "Error executing default steps")
		return UseResult(result)
	}

	// Try to build the Form HTML
	formHTML, err := form.Editor(schema, step.Form, object, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepAddModelObject.Get", "Error generating form"))
	}

	formHTML = WrapForm(builder.URL(), formHTML, step.Form.Encoding())

	// Wrap formHTML as a modal dialog
	if _, err := io.WriteString(buffer, formHTML); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepAddModelObject.Get", "Error writing form HTML to buffer"))
	}

	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepAddModelObject) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	// This finds/creates a new object in the builder
	factory := builder.factory()
	request := builder.request()
	object := builder.object()
	schema := builder.schema()

	// Execute any "default" steps so that the object is initialized
	result := Pipeline(step.Defaults).Post(factory, builder, buffer)

	if result.Halt {
		result.Error = derp.Wrap(result.Error, "build.StepAddModelObject.Post", "Error executing default steps")
		return UseResult(result)
	}

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.AddModelObject.Post", "Error parsing form data"))
	}

	// Try to set each path from the Form into the builder.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(object, key, value); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.AddModelObject.Post", "Error setting path value", key, value))
		}
	}

	// Save the object to the database
	if err := builder.service().ObjectSave(builder.session(), object, "Created"); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepAddModelObject.Post", "Error saving model object to database"))
	}

	// Success!
	return nil
}

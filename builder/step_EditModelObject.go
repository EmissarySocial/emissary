package builder

import (
	"io"
	"strings"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// StepEditModelObject is an action that can add new sub-streams to the domain.
type StepEditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepEditModelObject) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditModelObject.Get"

	schema := builder.schema()

	// Try to build the Form HTML
	result, err := form.Editor(schema, step.Form, builder.object(), builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error generating form"))
	}

	optionStrings := make([]string, 0, len(step.Options))
	for _, option := range step.Options {

		optionString := executeTemplate(option, builder)

		// Remove "delete" options from new objects.
		if builder.object().IsNew() && strings.HasPrefix(optionString, "delete:") {
			continue
		}

		// Otherwise, generate the text string for the option.
		optionStrings = append(optionStrings, optionString)
	}

	result = WrapForm(builder.URL(), result, optionStrings...)

	// nolint:errcheck
	io.WriteString(buffer, result)

	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepEditModelObject) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditModelObject.Post"

	// Get the request body
	body := mapof.NewAny()

	if err := bind(builder.request(), &body); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding request body"))
	}

	// Appy request body to the object (limited and validated by the form schema)
	stepForm := form.New(builder.schema(), step.Form)
	object := builder.object()

	if err := stepForm.SetAll(object, body, builder.lookupProvider()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error applying request body to model object", body))
	}

	// Save the object to the database
	if err := builder.service().ObjectSave(object, "Edited"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving model object to database"))
	}

	// Success!
	return nil
}

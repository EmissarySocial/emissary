package build

import (
	"io"
	"strings"
	"text/template"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepEditModelObject is an action that can add new sub-streams to the domain.
type StepEditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepEditModelObject) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditModelObject.Get"

	// Evaluate options to pass into the form.
	optionStrings := step.getOptions(builder)

	// Try to build the Form HTML
	schema := builder.schema()
	element := step.getForm(builder)
	result, err := form.Editor(schema, element, builder.object(), builder.lookupProvider(), optionStrings...)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Generating form"))
	}

	// Wrap the form layout in a <form> element
	result = WrapForm(builder.RelativeURL(), result, element.Encoding(), optionStrings...)

	// Write to the result buffer
	if _, err := io.WriteString(buffer, result); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditModelObject.Get", "Writing form HTML to buffer"))
	}

	// Retire in Cabo.
	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepEditModelObject) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditModelObject.Post"

	// Get the request data
	values, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Parsing form values"))
	}

	// RULE: Save through the options that Get drew with, because a field the options hide
	// must never be written, and a missing option reads as FALSE (BUG-334)
	stepForm := form.New(builder.schema(), step.getForm(builder), step.getOptions(builder)...)

	// Apply request body to the object (limited and validated by the form schema)
	object := builder.object()

	if err := stepForm.SetURLValues(object, values, builder.lookupProvider()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Applying request body to model object"))
	}

	// Success!
	return nil
}

// getOptions renders the step's option templates for this request, leaving out "delete" options on a new object.
func (step StepEditModelObject) getOptions(builder Builder) []string {

	result := make([]string, 0, len(step.Options))

	for _, option := range step.Options {

		optionString := executeTemplate(option, builder)

		// Remove "delete" options from new objects.
		if builder.object().IsNew() && strings.HasPrefix(optionString, "delete:") {
			continue
		}

		// Otherwise, generate the text string for the option.
		result = append(result, optionString)
	}

	return result
}

// getForm returns the form this step edits with, falling back to the Builder's own PropertyForm.
// Its option templates are rendered by the widgets, against the edited object.
func (step StepEditModelObject) getForm(builder Builder) form.Element {

	// If the step does not contain a form...
	if step.Form.IsEmpty() {
		// ...see if we can get the baked-in PropertyForm from the builder
		// (only the Domain builder, for now)
		if getter, ok := builder.(PropertyFormGetter); ok {
			return getter.PropertyForm()
		}
	}

	return step.Form
}

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

	schema := builder.schema()

	// Try to build the Form HTML
	element := step.getForm(builder)
	result, err := form.Editor(schema, element, builder.object(), builder.lookupProvider())

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

	result = WrapForm(builder.URL(), result, element.Encoding(), optionStrings...)

	// nolint:errcheck
	io.WriteString(buffer, result)

	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepEditModelObject) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditModelObject.Post"

	// Get the request data
	values, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form values"))
	}

	// Appy request body to the object (limited and validated by the form schema)
	stepForm := form.New(builder.schema(), step.getForm(builder))
	object := builder.object()

	if err := stepForm.SetURLValues(object, values, builder.lookupProvider()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error applying request body to model object"))
	}

	// Success!
	return nil
}

func (step StepEditModelObject) getForm(builder Builder) form.Element {

	form := step.Form

	// If the step does not contain a form...
	if form.IsEmpty() {
		// ...see if we can get the baked-in PropertyForm from the builder
		// (only the Domain builder, for now)
		if getter, ok := builder.(PropertyFormGetter); ok {
			form = getter.PropertyForm()
		}
	}

	// replace all template values in the form (and its children)
	result := step.executeOptionTemplates(builder, form)

	return result
}

func (step StepEditModelObject) executeOptionTemplates(builder Builder, element form.Element) form.Element {

	// Recursively scan all child elements
	for index, child := range element.Children {
		element.Children[index] = step.executeOptionTemplates(builder, child)
	}

	// Scan all options in this element
	for key, value := range element.Options {
		switch typed := value.(type) {
		case string:
			if strings.Contains(typed, "{{") {
				if template, err := template.New("option").Parse(typed); err == nil {
					element.Options[key] = executeTemplate(template, builder)
				}
			}
		}
	}

	// Return the modified element
	return element
}

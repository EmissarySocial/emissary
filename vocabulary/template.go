package vocabulary

import (
	"html/template"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
	"github.com/benpate/schema"
)

type TemplateArgs struct {
	Form   form.Form
	Schema schema.Element
	Value  interface{}
}

func RegisterTemplate(library form.Library, name string, html string) error {

	// Parse the HTML template
	t, err := template.New(name).Parse(html)

	// Handle errors
	if err != nil {
		return derp.Wrap(err, "form.TemplateWidget", "Error registering Template", name, html)
	}

	// Register the template in the provided form registry
	library.Register(name, func(form form.Form, schema schema.Schema, value interface{}, builder *strings.Builder) error {

		// Get the path to the value
		p := path.New(form.Path)

		args := TemplateArgs{
			Form: form,
		}

		if schemaValue, err := schema.Path(p); err != nil {
			args.Schema = schemaValue
		}

		// Try to get the value from the data provided
		if formValue, err := p.Get(value); err != nil {
			args.Value = formValue
		}

		if err := t.Execute(builder, args); err != nil {
			return derp.Wrap(err, "TemplateWidget", "Error executing template", name, form)
		}

		return nil
	})

	// Success!
	return nil
}

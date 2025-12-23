package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
)

// StepEditTemplate is a Step that lets users choose their profile template(s)
type StepEditTemplate struct {
	Title string
	Paths []string
}

// Get displays a form where users can select their profile template(s)
func (step StepEditTemplate) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	schema := builder.schema()

	form := form.New(schema, form.Element{
		Type:  "layout-vertical",
		Label: step.Title,
		Children: slice.Map(step.Paths, func(path string) form.Element {

			// Create the form element for the requested field
			return form.Element{
				Type:  "select",
				Label: step.fieldLabel(path),
				Path:  path,
				Options: mapof.Any{
					"enum": step.listTemplates(builder, path),
				},
			}
		}),
	})

	// Write the form to HTML
	h := html.New()
	if err := form.BuildEditor(builder.object(), nil, h); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditTemplate.Get", "Error building form"))
	}

	result := WrapForm(builder.URL(), h.String(), form.Encoding())

	// Write the HTML to the buffer
	if _, err := buffer.Write([]byte(result)); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditTemplate.Get", "Error writing HTML"))
	}

	return Continue()
}

// Post updates a User's profile template(s)
func (step StepEditTemplate) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditTemplate.Post"

	schema := builder.schema()
	object := builder.object()

	// Collect inputs from the form
	transaction, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form"))
	}

	// Multiple Templates may be specified.  So, for each new value...
	for _, path := range step.Paths {

		// Scan all allowed records
		if newTemplateID := transaction.Get(path); step.isTemplateAllowed(builder, path, newTemplateID) {
			if err := schema.Set(object, path, newTemplateID); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error setting template", path))
			}
		}
	}

	// Success!
	return Continue()
}

func (step StepEditTemplate) fieldLabel(value string) string {

	switch value {

	case "templateId":
		return "Template"

	case "inboxTemplate":
		return "Profile"

	case "outboxTemplate":
		return "Outbox"
	}

	return ""
}

func (step StepEditTemplate) isTemplateAllowed(builder Builder, path string, templateID string) bool {

	allowedTemplates := step.listTemplates(builder, path)

	for _, allowedTemplate := range allowedTemplates {
		if allowedTemplate.Value == templateID {
			return true
		}
	}

	return false
}

func (step StepEditTemplate) listTemplates(builder Builder, path string) []form.LookupCode {

	switch path {

	case "templateId":
		if stream, ok := builder.object().(*model.Stream); ok {

			templateService := builder.factory().Template()
			parentTemplateID := stream.ParentTemplateID

			if parentTemplate, err := templateService.Load(parentTemplateID); err == nil {
				return templateService.ListByContainer(parentTemplate.TemplateRole)
			}
		}

	case "inboxTemplate":
		return builder.factory().Template().ListByTemplateRole("user-inbox")

	case "outboxTemplate":
		return builder.factory().Template().ListByTemplateRole("user-outbox")
	}

	return make([]form.LookupCode, 0)
}

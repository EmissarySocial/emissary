package builder

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepSetTemplate represents an action-step that can delete a Stream from the Domain
type StepSetTemplate struct {
	Paths map[string]*template.Template
}

// Get displays a customizable confirmation form for the delete
func (step StepSetTemplate) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return Continue()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepSetTemplate) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetTemplate.Post"

	templateService := builder.factory().Template()
	schema := builder.schema()

	// Multiple Templates may be specified.  So, for each new value...
	for path, template := range step.Paths {

		// Find the original TemplateID
		originalTemplateID, err := schema.Get(builder.object, path)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error getting original template", path))
		}

		// Find the new TemplateID
		newTemplateID := executeTemplate(template, builder)

		// Confirm that the new Template is the same as the old one.
		if !templateService.MatchesRole(convert.String(originalTemplateID), newTemplateID) {
			return Halt().WithError(derp.NewForbiddenError(location, "Template does not match role", path))
		}

		// Update the record
		if err := schema.Set(builder.object, path, newTemplateID); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error setting template", path))
		}
	}

	// Success!
	return Continue()
}

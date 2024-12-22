package step

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// EditContent is a Step that can edit/update Container in a streamDraft.
type EditContent struct {
	Filename  string
	Fieldname string
	Format    string
}

func NewEditContent(stepInfo mapof.Any) (EditContent, error) {

	// Validate the step configuration
	if err := StepEditContentSchema().Validate(stepInfo); err != nil {
		return EditContent{}, err
	}

	// Create the new "edit-content" step
	return EditContent{
		Filename:  first(stepInfo.GetString("file"), stepInfo.GetString("actionId")),
		Fieldname: first(stepInfo.GetString("field"), "content"),
		Format:    first(stepInfo.GetString("format"), "editorjs"),
	}, nil
}

// StepEditContentSchema returns a validating schema for the EditContent step
func StepEditContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"filename": schema.String{},
			"format": schema.String{
				Required: true,
				Enum: []string{
					"EDITORJS",
					"HTML",
					"MARKDOWN",
					"TEXT",
				},
			},
		},
	}
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step EditContent) AmStep() {}

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

// Name returns the name of the step, which is used in debugging.
func (step EditContent) Name() string {
	return "edit-content"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditContent) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditContent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditContent) RequiredRoles() []string {
	return []string{}
}

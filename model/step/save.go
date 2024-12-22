package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Save is a Step that can save changes to any object
type Save struct {
	Comment *template.Template
	Method  string
}

// NewSave returns a fully initialized Save object
func NewSave(stepInfo mapof.Any) (Save, error) {

	// Validate the step configuration
	if err := StepSaveSchema().Validate(stepInfo); err != nil {
		return Save{}, derp.Wrap(err, "model.step.NewSave", "Invalid step configuration", stepInfo)
	}

	// Get the "comment" template
	comment, err := template.New("").Parse(stepInfo.GetString("comment"))

	if err != nil {
		return Save{}, derp.Wrap(err, "model.step.NewSave", "Error parsing comment template")
	}

	// Create the new Save step
	return Save{
		Comment: comment,
		Method:  first(stepInfo.GetString("method"), "post"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Save) AmStep() {}

// StepSaveSchema returns a validating schema for the EditContent step
func StepSaveSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"comment": schema.String{MaxLength: 1024, Format: "text"},
			"method":  schema.String{Enum: []string{"get", "post", "both"}, Default: "both"},
		},
	}
}

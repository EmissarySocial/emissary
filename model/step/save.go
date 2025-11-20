package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Save is a Step that can save changes to any object
type Save struct {
	Comment *template.Template
	Method  string
	OnError []Step
}

// NewSave returns a fully initialized Save object
func NewSave(stepInfo mapof.Any) (Save, error) {

	const location = "model.step.NewSave"

	// Validate the step configuration
	if err := StepSaveSchema().Validate(stepInfo); err != nil {
		return Save{}, derp.Wrap(err, location, "Invalid step configuration", stepInfo)
	}

	// Get the "comment" template
	comment, err := template.New("").Parse(stepInfo.GetString("comment"))

	if err != nil {
		return Save{}, derp.Wrap(err, location, "Error parsing comment template")
	}

	onError, err := NewPipeline(convert.SliceOfMap(stepInfo["on-error"]))

	if err != nil {
		return Save{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	// Create the new Save step
	return Save{
		Comment: comment,
		Method:  first(stepInfo.GetString("method"), "post"),
		OnError: onError,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Save) Name() string {
	return "save"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Save) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Save) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Save) RequiredRoles() []string {
	return []string{}
}

// StepSaveSchema returns a validating schema for the EditContent step
func StepSaveSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"comment": schema.String{MaxLength: 1024, Format: "text"},
			"method":  schema.String{Enum: []string{"get", "post", "both"}, Default: "both"},
		},
	}
}

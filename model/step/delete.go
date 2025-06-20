package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Delete is a Step that can delete a Stream from the Domain
type Delete struct {
	Title   *template.Template
	Message *template.Template
	Submit  string
	Method  string
}

// NewDelete returns a fully populated Delete object
func NewDelete(stepInfo mapof.Any) (Delete, error) {

	const location = "model.step.NewDelete"

	// Validate the step configuration
	if err := StepDeleteSchema().Validate(stepInfo); err != nil {
		return Delete{}, derp.Wrap(err, location, "Invalid step configuration", stepInfo)
	}

	// Create the "title" template
	titleTemplate, err := template.New("").Parse(first(stepInfo.GetString("title"), "Delete '{{.Label}}'?"))

	if err != nil {
		return Delete{}, derp.Wrap(err, location, "Invalid 'title' template", stepInfo)
	}

	// Create the "message" template
	messageTemplate, err := template.New("").Parse(first(stepInfo.GetString("message"), "Are you sure you want to delete {{.Label}}? There is NO UNDO."))

	if err != nil {
		return Delete{}, derp.Wrap(err, location, "Invalid 'message' template", stepInfo)
	}

	return Delete{
		Title:   titleTemplate,
		Message: messageTemplate,
		Submit:  first(stepInfo.GetString("submit"), "Delete"),
		Method:  first(stepInfo.GetString("method"), "both"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Delete) Name() string {
	return "delete"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Delete) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Delete) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Delete) RequiredRoles() []string {
	return []string{}
}

// StepDeleteSchema returns a validating schema for the EditContent step
func StepDeleteSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"title":   schema.String{MaxLength: 128, Format: "text"},
			"message": schema.String{MaxLength: 512, Format: "text"},
			"submit":  schema.String{MaxLength: 32, Format: "text"},
			"method":  schema.String{Enum: []string{"get", "post", "both"}, Default: "both"},
		},
	}
}

package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// RequirePassword is a Step that requires the user to enter their password before performing a sensitive action
type RequirePassword struct {
	Title       *template.Template
	Message     *template.Template
	Submit      string
	SubmitClass string
	Cancel      string
}

// NewRequirePassword returns a fully populated RequirePassword object
func NewRequirePassword(stepInfo mapof.Any) (RequirePassword, error) {

	const location = "model.step.NewRequirePassword"

	// Validate the step configuration
	if err := StepRequirePasswordSchema().Validate(stepInfo); err != nil {
		return RequirePassword{}, derp.Wrap(err, location, "Invalid step configuration", stepInfo)
	}

	// Create the "title" template
	titleTemplate, err := template.New("").Parse(first(stepInfo.GetString("title"), "Delete '{{.Label}}'?"))

	if err != nil {
		return RequirePassword{}, derp.Wrap(err, location, "Invalid 'title' template", stepInfo)
	}

	// Create the "message" template
	messageTemplate, err := template.New("").Parse(first(stepInfo.GetString("message"), "Are you sure you want to delete {{.Label}}? There is NO UNDO."))

	if err != nil {
		return RequirePassword{}, derp.Wrap(err, location, "Invalid 'message' template", stepInfo)
	}

	return RequirePassword{
		Title:       titleTemplate,
		Message:     messageTemplate,
		Submit:      first(stepInfo.GetString("submit"), "Confirm"),
		SubmitClass: first(stepInfo.GetString("submitClass"), "warning"),
		Cancel:      first(stepInfo.GetString("cancel"), "Cancel"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step RequirePassword) Name() string {
	return "requirePassword"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step RequirePassword) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step RequirePassword) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step RequirePassword) RequiredRoles() []string {
	return []string{}
}

// StepRequirePasswordSchema returns a validating schema for the RequirePassword step
func StepRequirePasswordSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"title":   schema.String{MaxLength: 128, Format: "text"},
			"message": schema.String{MaxLength: 256, Format: "text"},
			"submit":  schema.String{MaxLength: 32, Format: "text"},
			"cancel":  schema.String{MaxLength: 32, Format: "text"},
		},
	}
}

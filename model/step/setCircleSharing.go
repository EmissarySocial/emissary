package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetCircleSharing represents an action that can edit a top-level folder in the Domain
type SetCircleSharing struct {
	Title   string
	Message string
	Button  string
	Role    string
}

// NewSetCircleSharing returns a fully parsed SetCircleSharing object
func NewSetCircleSharing(stepInfo mapof.Any) (SetCircleSharing, error) {

	role := stepInfo.GetString("role")

	if role == "" {
		return SetCircleSharing{}, derp.ValidationError("Role is required")
	}

	return SetCircleSharing{
		Title:   first(stepInfo.GetString("title"), "Sharing Settings"),
		Message: first(stepInfo.GetString("message"), "Public Settings"),
		Button:  first(stepInfo.GetString("button"), "Save Changes"),
		Role:    first(stepInfo.GetString("role"), "editor"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetCircleSharing) Name() string {
	return "set-circle-sharing"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetCircleSharing) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetCircleSharing) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetCircleSharing) RequiredRoles() []string {
	return []string{step.Role}
}

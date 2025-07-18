package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetSimpleSharing represents an action that can edit a top-level folder in the Domain
type SetSimpleSharing struct {
	Title   string
	Message string
	Role    string
}

// NewSetSimpleSharing returns a fully parsed SetSimpleSharing object
func NewSetSimpleSharing(stepInfo mapof.Any) (SetSimpleSharing, error) {

	// Confirm that the role is defined
	role := stepInfo.GetString("role")

	if role == "" {
		return SetSimpleSharing{}, derp.BadRequestError("step.NewSetSimpleSharing", "Role is required")
	}

	return SetSimpleSharing{
		Title:   first(stepInfo.GetString("title"), "Sharing Settings"),
		Message: first(stepInfo.GetString("message"), ""),
		Role:    role,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetSimpleSharing) Name() string {
	return "set-simple-sharing"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetSimpleSharing) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetSimpleSharing) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetSimpleSharing) RequiredRoles() []string {
	return []string{step.Role}
}

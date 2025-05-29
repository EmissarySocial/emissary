package step

import (
	"github.com/benpate/rosetta/mapof"
)

// AsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type AsConfirmation struct {
	Icon    string
	Title   string
	Message string
	Submit  string
}

// NewAsConfirmation returns a fully initialized AsConfirmation object
func NewAsConfirmation(stepInfo mapof.Any) (AsConfirmation, error) {

	return AsConfirmation{
		Icon:    stepInfo.GetString("icon"),
		Title:   stepInfo.GetString("title"),
		Message: stepInfo.GetString("message"),
		Submit:  first(stepInfo.GetString("submit"), "Continue"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AsConfirmation) Name() string {
	return "as-confirmation"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AsConfirmation) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step AsConfirmation) RequiredRoles() []string {
	return []string{}
}

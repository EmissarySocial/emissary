package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SendEmail is a Step that can send a named email to a user
type SendEmail struct {
	Email string
}

// NewSendEmail returns a fully initialized SendEmail object
func NewSendEmail(stepInfo mapof.Any) (SendEmail, error) {
	return SendEmail{
		Email: stepInfo.GetString("email"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SendEmail) Name() string {
	return "send-email"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SendEmail) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SendEmail) RequiredRoles() []string {
	return []string{}
}

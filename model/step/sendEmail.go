package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SendEmail represents an action-step that can send a named email to a user
type SendEmail struct {
	Email string
}

// NewSendEmail returns a fully initialized SendEmail object
func NewSendEmail(stepInfo mapof.Any) (SendEmail, error) {
	return SendEmail{
		Email: stepInfo.GetString("email"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SendEmail) AmStep() {}

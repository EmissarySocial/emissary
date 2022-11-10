package render

import (
	"io"
)

// StepDeleteSubscription is an action that can delete a subscription for the current user.
type StepDeleteSubscription struct {
}

func (step StepDeleteSubscription) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepDeleteSubscription) UseGlobalWrapper() bool {
	return false
}

func (step StepDeleteSubscription) Post(renderer Renderer) error {
	return nil
}

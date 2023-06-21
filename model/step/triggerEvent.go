package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// TriggerEvent represents an action-step that forwards the user to a new page.
type TriggerEvent struct {
	Event string
	Value string
}

// NewTriggerEvent returns a fully initialized TriggerEvent object
func NewTriggerEvent(stepInfo mapof.Any) (TriggerEvent, error) {

	return TriggerEvent{
		Event: stepInfo.GetString("event"),
		Value: first.String(stepInfo.GetString("value"), "true"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step TriggerEvent) AmStep() {}

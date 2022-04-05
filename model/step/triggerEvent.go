package step

import (
	"github.com/benpate/datatype"
)

// TriggerEvent represents an action-step that forwards the user to a new page.
type TriggerEvent struct {
	Event string
	Data  string
}

// NewTriggerEvent returns a fully initialized TriggerEvent object
func NewTriggerEvent(stepInfo datatype.Map) (TriggerEvent, error) {

	return TriggerEvent{
		Event: stepInfo.GetString("event"),
		Data:  stepInfo.GetString("data"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step TriggerEvent) AmStep() {}

package step

import (
	"encoding/json"

	"github.com/benpate/rosetta/maps"
)

// TriggerEvent represents an action-step that forwards the user to a new page.
type TriggerEvent struct {
	Event string
}

// NewTriggerEvent returns a fully initialized TriggerEvent object
func NewTriggerEvent(stepInfo maps.Map) (TriggerEvent, error) {

	eventData := getValue(stepInfo.GetInterface("event"))
	buffer, _ := json.Marshal(eventData)
	eventString := string(buffer)

	return TriggerEvent{
		Event: eventString,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step TriggerEvent) AmStep() {}

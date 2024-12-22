package step

import (
	"github.com/benpate/rosetta/mapof"
)

// RemoveEvent is a Step that removes an HX-Trigger event from the HTTP result
type RemoveEvent struct {
	Event string
}

// NewRemoveEvent returns a fully initialized RemoveEvent object
func NewRemoveEvent(stepInfo mapof.Any) (RemoveEvent, error) {

	return RemoveEvent{
		Event: stepInfo.GetString("event"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step RemoveEvent) AmStep() {}

package step

import "github.com/benpate/rosetta/maps"

// Publish represents an action-step that can update a stream's PublishDate with the current time.
type Publish struct {
	Role string
}

// NewPublish returns a fully initialized Publish object
func NewPublish(stepInfo maps.Map) (Publish, error) {
	return Publish{
		Role: getValue(stepInfo.GetString("role")),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Publish) AmStep() {}

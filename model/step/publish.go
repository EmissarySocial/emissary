package step

import "github.com/benpate/rosetta/mapof"

// Publish represents an action-step that can update a stream's PublishDate with the current time.
type Publish struct {
	Role string
}

// NewPublish returns a fully initialized Publish object
func NewPublish(stepInfo mapof.Any) (Publish, error) {
	return Publish{
		Role: stepInfo.GetString("role"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Publish) AmStep() {}

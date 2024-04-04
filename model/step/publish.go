package step

import "github.com/benpate/rosetta/mapof"

// Publish represents an action-step that can update a stream's PublishDate with the current time.
type Publish struct {
	Outbox bool // If TRUE, also send updates to this User's outbox.
}

// NewPublish returns a fully initialized Publish object
func NewPublish(stepInfo mapof.Any) (Publish, error) {
	result := Publish{
		Outbox: stepInfo.GetBool("outbox"),
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Publish) AmStep() {}

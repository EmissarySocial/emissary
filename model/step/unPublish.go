package step

import "github.com/benpate/rosetta/mapof"

// UnPublish represents an action-step that can update a stream's UnPublishDate with the current time.
type UnPublish struct {
	Outbox bool
}

// NewUnPublish returns a fully initialized UnPublish object
func NewUnPublish(stepInfo mapof.Any) (UnPublish, error) {
	return UnPublish{
		Outbox: stepInfo.GetBool("outbox"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step UnPublish) AmStep() {}

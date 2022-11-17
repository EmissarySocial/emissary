package step

import "github.com/benpate/rosetta/maps"

// UnPublish represents an action-step that can update a stream's UnPublishDate with the current time.
type UnPublish struct {
}

// NewUnPublish returns a fully initialized UnPublish object
func NewUnPublish(stepInfo maps.Map) (UnPublish, error) {
	return UnPublish{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step UnPublish) AmStep() {}

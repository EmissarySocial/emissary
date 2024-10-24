package step

import "github.com/benpate/rosetta/mapof"

// SaveAndPublish represents an action-step that can update a stream's SaveAndPublishDate with the current time.
type SaveAndPublish struct {
	Outbox bool // If TRUE, also send updates to this User's outbox.
}

// NewSaveAndPublish returns a fully initialized SaveAndPublish object
func NewSaveAndPublish(stepInfo mapof.Any) (SaveAndPublish, error) {
	result := SaveAndPublish{
		Outbox: stepInfo.GetBool("outbox"),
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SaveAndPublish) AmStep() {}

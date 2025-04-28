package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SaveAndPublish is a Step that can update a stream's SaveAndPublishDate with the current time.
type SaveAndPublish struct {
	Outbox    bool // If TRUE, also send updates to this User's outbox.
	Republish bool // If TRUE, republishes this stream to syndication targets.
}

// NewSaveAndPublish returns a fully initialized SaveAndPublish object
func NewSaveAndPublish(stepInfo mapof.Any) (SaveAndPublish, error) {

	result := SaveAndPublish{
		Outbox:    stepInfo.GetBool("outbox"),
		Republish: stepInfo.GetBool("republish"),
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SaveAndPublish) AmStep() {}

package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// StreamPromoteDraft represents a pipeline-step that can copy the Container from a StreamDraft into its corresponding Stream
type StreamPromoteDraft struct {
	StateID string
}

func NewStreamPromoteDraft(stepInfo mapof.Any) (StreamPromoteDraft, error) {
	return StreamPromoteDraft{
		StateID: first.String(stepInfo.GetString("state"), "published"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step StreamPromoteDraft) AmStep() {}

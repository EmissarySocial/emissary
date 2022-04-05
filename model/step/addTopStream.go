package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// AddTopStream represents an action that can create top-level folders in the Domain
type AddTopStream struct {
	TemplateIDs   []string // List of valid templateIds that the new top-level stream could be
	WithNewStream []Step   // Pipeline of steps to take on the newly-created stream
}

// NewAddTopStream returns a fully parsed AddTopStream object
func NewAddTopStream(stepInfo datatype.Map) (AddTopStream, error) {

	withNewStream, err := NewPipeline(stepInfo.GetSliceOfMap("with-new-stream"))

	if err != nil {
		return AddTopStream{}, derp.Wrap(err, "model.step.AddTopStream", "Invalid 'with-new-stream", stepInfo)
	}

	return AddTopStream{
		TemplateIDs:   stepInfo.GetSliceOfString("templateIds"),
		WithNewStream: withNewStream,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddTopStream) AmStep() {}

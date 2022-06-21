package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
)

// AddTopStream represents an action that can create top-level folders in the Domain
type AddTopStream struct {
	Title         string
	TemplateIDs   []string // List of valid templateIds that the new top-level stream could be
	WithNewStream []Step   // Pipeline of steps to take on the newly-created stream
}

// NewAddTopStream returns a fully parsed AddTopStream object
func NewAddTopStream(stepInfo maps.Map) (AddTopStream, error) {

	withNewStream, err := NewPipeline(stepInfo.GetSliceOfMap("with-new-stream"))

	if err != nil {
		return AddTopStream{}, derp.Wrap(err, "model.step.AddTopStream", "Invalid 'with-new-stream", stepInfo)
	}

	return AddTopStream{
		Title:         first.String(stepInfo.GetString("title"), "Add a Stream"),
		TemplateIDs:   stepInfo.GetSliceOfString("templateIds"),
		WithNewStream: withNewStream,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddTopStream) AmStep() {}

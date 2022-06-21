package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
)

// AddChildStream is an action that can add new sub-streams to the domain.
type AddChildStream struct {
	Title       string
	TemplateIDs []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	View        string   // If present, use this HTML template as a custom "create" page.  If missing, a default modal pop-up is used.
	WithChild   []Step   // List of steps to take on the newly created child record on POST.
}

// NewAddChildStream returns a fully initialized AddChildStream record
func NewAddChildStream(stepInfo maps.Map) (AddChildStream, error) {

	withChild, err := NewPipeline(stepInfo.GetSliceOfMap("with-child"))

	if err != nil {
		return AddChildStream{}, derp.Wrap(err, "model.setp.NewAddChildStream", "Error parsing with-child")
	}

	return AddChildStream{
		Title:       first.String(stepInfo.GetString("title"), "Add a Stream"),
		View:        stepInfo.GetString("view"),
		TemplateIDs: stepInfo.GetSliceOfString("template"),
		WithChild:   withChild,
	}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step AddChildStream) AmStep() {}

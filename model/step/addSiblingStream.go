package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// AddSiblingStream is an action that can add new sub-streams to the domain.
type AddSiblingStream struct {
	TemplateIDs []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	View        string   // If present, use this HTML template as a custom "create" page.  If missing, a default modal pop-up is used.
	WithSibling []Step   // List of steps to take on the newly created sibling record on POST.
}

// NewAddSiblingStream returns a fully initialized AddSiblingStream record
func NewAddSiblingStream(stepInfo datatype.Map) (AddSiblingStream, error) {

	withSibling, err := NewPipeline(stepInfo.GetSliceOfMap("with-sibling"))

	if err != nil {
		return AddSiblingStream{}, derp.Wrap(err, "model.step.NewStepAddWithSibling", "Invalid 'with-sibling", stepInfo)
	}

	return AddSiblingStream{
		View:        stepInfo.GetString("view"),
		TemplateIDs: stepInfo.GetSliceOfString("template"),
		WithSibling: withSibling,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddSiblingStream) AmStep() {}

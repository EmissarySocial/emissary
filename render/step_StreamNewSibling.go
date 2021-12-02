package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepNewSibling is an action that can add new sub-streams to the domain.
type StepNewSibling struct {
	streamService *service.Stream
	withChild     []datatype.Map
}

// NewStepNewSibling returns a fully initialized StepNewSibling record
func NewStepNewSibling(streamService *service.Stream, stepInfo datatype.Map) StepNewSibling {
	return StepNewSibling{
		streamService: streamService,
		withChild:     stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepNewSibling) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepNewSibling) Post(buffer io.Writer, renderer *Renderer) error {

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(renderer.ctx)

	// Set Default Values
	child.ParentID = renderer.stream.ParentID
	child.StateID = "default"
	child.AuthorID = authorization.UserID

	if err := DoPipeline(renderer, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewSibling", "Error running post-create steps")
	}

	return nil
}

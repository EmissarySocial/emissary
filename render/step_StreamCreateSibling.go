package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepCreateSibling is an action that can add new sub-streams to the domain.
type StepCreateSibling struct {
	streamService *service.Stream
	withChild     []datatype.Map
}

// NewStepCreateSibling returns a fully initialized StepCreateSibling record
func NewStepCreateSibling(streamService *service.Stream, stepInfo datatype.Map) StepCreateSibling {
	return StepCreateSibling{
		streamService: streamService,
		withChild:     stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepCreateSibling) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepCreateSibling) Post(buffer io.Writer, renderer *Renderer) error {

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(renderer.ctx)

	// Set Default Values
	child.ParentID = renderer.stream.ParentID
	child.StateID = "default"
	child.AuthorID = authorization.UserID

	if err := DoPipeline(renderer, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepCreateSibling", "Error running post-create steps")
	}

	return nil
}

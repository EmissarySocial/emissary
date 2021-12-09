package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// StepNewSibling is an action that can add new sub-streams to the domain.
type StepNewSibling struct {
	streamService *service.Stream
	template      []string
	withChild     []datatype.Map
}

// NewStepNewSibling returns a fully initialized StepNewSibling record
func NewStepNewSibling(streamService *service.Stream, stepInfo datatype.Map) StepNewSibling {
	return StepNewSibling{
		streamService: streamService,
		template:      stepInfo.GetSliceOfString("template"),
		withChild:     stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepNewSibling) Get(buffer io.Writer, renderer *Stream) error {
	return nil
}

func (step StepNewSibling) Post(buffer io.Writer, renderer *Stream) error {

	templateID := renderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.template) > 0 {

		if templateID == "" {
			templateID = step.template[0]
		} else if !compare.Contains(step.template, templateID) {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepNewChild.Post", "Cannot create new template of this kind", templateID)
		}
	}

	// Create new sibling stream
	sibling, _, err := step.streamService.NewSibling(renderer.stream, templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewSibling", "Error creating sibling stream", templateID)
	}

	// Set Default Values
	authorization := getAuthorization(renderer.ctx)
	sibling.AuthorID = authorization.UserID
	siblingStream, err := renderer.newStream(&sibling, "edit")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", sibling)
	}

	if err := DoPipeline(&siblingStream, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewSibling", "Error running post-create steps")
	}

	return nil
}

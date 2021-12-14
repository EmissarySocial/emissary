package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// StepTopLevelCreate represents an action that can create top-level folders in the Domain
type StepTopLevelCreate struct {
	streamService *service.Stream
	template      []string
}

// NewStepTopLevelCreate returns a fully parsed StepTopLevelCreate object
func NewStepTopLevelCreate(streamService *service.Stream, config datatype.Map) StepTopLevelCreate {

	return StepTopLevelCreate{
		streamService: streamService,
		template:      config.GetSliceOfString("template"),
	}
}

func (step StepTopLevelCreate) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

func (step StepTopLevelCreate) Post(buffer io.Writer, renderer Renderer) error {

	domainRenderer := renderer.(*Domain)
	templateID := domainRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.template) > 0 {

		if templateID == "" {
			templateID = step.template[0]
		} else if !compare.Contains(step.template, templateID) {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepNewChild.Post", "Cannot create new template of this kind", templateID)
		}
	}

	child, template, err := step.streamService.NewTopLevel(templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepTopLevelCreate.Post", "Error creating TopLevel stream", templateID)
	}

	// Set stream defaults
	authorization := getAuthorization(domainRenderer.ctx)
	child.AuthorID = authorization.UserID
	childStream, err := NewStream(domainRenderer.factory, domainRenderer.ctx, template, child, "view")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(domainRenderer.factory, &childStream, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute 'init' action on child")
		}
	}

	return nil
}

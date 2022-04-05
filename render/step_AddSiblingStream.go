package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
)

// StepAddSiblingStream is an action that can add new sub-streams to the domain.
type StepAddSiblingStream struct {
	templateIDs []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	view        string   // If present, use this HTML template as a custom "create" page.  If missing, a default modal pop-up is used.
	withSibling Pipeline // List of steps to take on the newly created sibling record on POST.

	BaseStep
}

// NewStepAddSiblingStream returns a fully initialized StepAddSiblingStream record
func NewStepAddSiblingStream(stepInfo datatype.Map) (StepAddSiblingStream, error) {

	withSibling, err := NewPipeline(stepInfo.GetSliceOfMap("with-sibling"))

	if err != nil {
		return StepAddSiblingStream{}, derp.Wrap(err, "render.NewStepAddWithSibling", "Invalid 'with-sibling", stepInfo)
	}

	return StepAddSiblingStream{
		view:        stepInfo.GetString("view"),
		templateIDs: stepInfo.GetSliceOfString("template"),
		withSibling: withSibling,
	}, nil
}

func (step StepAddSiblingStream) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	// This can only be used on a Stream Renderer
	streamRenderer := renderer.(*Stream)

	// If a view has been specified, then use it to render a "create" page
	if step.view != "" {

		if err := renderer.executeTemplate(buffer, step.view, renderer); err != nil {
			return derp.Wrap(err, "whisper.render.StepViewHTML.Get", "Error executing template")
		}

		return nil
	}

	// Fall through to displaying the default modal
	modalAddStream(renderer.context().Response(), factory.Template(), buffer, streamRenderer.URL(), streamRenderer.TemplateID(), step.templateIDs)

	return nil
}

func (step StepAddSiblingStream) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	// Collect prerequisites
	streamRenderer := renderer.(*Stream)
	context := streamRenderer.context()
	sibling := streamRenderer.stream

	// New Stream is assumed to use the same Template as the current Stream
	templateID := streamRenderer.template.TemplateID

	// But if there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {
		if paramTemplateID := streamRenderer.ctx.QueryParam("templateId"); paramTemplateID != "" {
			if compare.Contains(step.templateIDs, paramTemplateID) {
				templateID = paramTemplateID
			}
		}
	}

	// Try to load parent Stream to validate data
	parent := model.NewStream()
	if err := factory.Stream().LoadParent(sibling, &parent); err != nil {
		return derp.Wrap(err, "service.Stream.NewSiblling", "Error loading parent Stream")
	}

	// Try to load parent's Template
	template, err := factory.Template().Load(templateID)

	if err != nil {
		return derp.Wrap(err, "service.Stream.NewTopLevel", "Cannot find template")
	}

	// Verify that the new child can be placed underneath the parent
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.New(derp.CodeInternalError, "service.Stream.NewTopLevel", "Template cannot be placed at top level", templateID)
	}

	// Create the new Stream
	stream := model.NewStream()
	stream.ParentID = parent.StreamID
	stream.ParentIDs = append(parent.ParentIDs, parent.StreamID)
	stream.TemplateID = templateID

	// TODO: sort order?

	return finalizeAddStream(buffer, renderer.factory(), context, &stream, template, step.withSibling)
}

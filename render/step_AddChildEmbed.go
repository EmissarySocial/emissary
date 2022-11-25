package render

import (
	"io"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
)

// StepAddChildEmbed is an action that can add new sub-streams to the domain.
type StepAddChildEmbed struct {
	TemplateIDs []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
}

func (step StepAddChildEmbed) common(renderer Renderer) ([]form.LookupCode, Renderer, error) {

	const location = "render.StepAddChildEmbed.common"

	// Get prerequisites
	factory := renderer.factory()
	context := renderer.context()
	templateService := factory.Template()

	// Query all eligible templates
	templates := templateService.ListByContainerLimited(renderer.templateRole(), step.TemplateIDs)

	if len(templates) == 0 {
		return nil, nil, derp.NewBadRequestError(location, "No child templates available for this stream", renderer.templateRole())
	}

	// Find the "selected" template
	templateID := step.getBestTemplate(templates, context.QueryParam("templateId"))
	streamService := factory.Stream()

	// Create a new child stream
	child, template, err := streamService.New(renderer.TopLevelID(), renderer.objectID(), templateID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error creating new child stream")
	}

	// Create a new child renderer
	childRenderer, err := NewStream(factory, context, template, &child, "create")

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error creating new child stream renderer")
	}

	return templates, childRenderer, nil

}

func (step StepAddChildEmbed) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddChildEmbed.Get"

	templates, childRenderer, err := step.common(renderer)

	if err != nil {
		return derp.Wrap(err, location, "Error getting common data")
	}

	iconService := renderer.factory().Icons()
	b := html.New()

	path := renderer.context().Path()
	selectedTemplateID := childRenderer.template().TemplateID

	if len(templates) > 1 {
		b.Div()
		for _, template := range templates {

			b.A("").Data("hx-get", path+"?templateId="+template.Value).EndBracket()
			iconService.Write(template.Icon, b)

			if selectedTemplateID == template.Value {
				b.WriteString(" " + template.Label)
			}

			b.Close()
		}
		b.Close()
	}

	widgetHTML, err := childRenderer.Render()

	if err != nil {
		return derp.Wrap(err, location, "Error rendering new child stream")
	}

	b.WriteString(string(widgetHTML))

	return renderer.context().HTML(http.StatusOK, b.String())
}

func (step StepAddChildEmbed) UseGlobalWrapper() bool {
	return false
}

func (step StepAddChildEmbed) Post(renderer Renderer) error {

	const location = "render.StepAddChildEmbed.Post"

	// Get pre-requisites
	factory := renderer.factory()
	responseWriter := renderer.context().Response()
	_, childRenderer, err := step.common(renderer)

	if err != nil {
		return derp.Wrap(err, location, "Error getting common data")
	}

	// Get the "create" action from the template
	action := childRenderer.template().Action("create")

	if action == nil {
		return derp.NewInternalError(location, "No 'create' action found in template", childRenderer.template().TemplateID)
	}

	// Execute the "CREATE" pipeline to save the new stream.
	if err := Pipeline(action.Steps).Execute(factory, childRenderer, responseWriter, ActionMethodPost); err != nil {
		return derp.Wrap(err, location, "Error executing pipeline")
	}

	// Done.
	return nil

	/*
		// Collect prerequisites
		factory := renderer.factory()
		context := renderer.context()
		templateID := context.QueryParam("templateId")

		// If there is a list of eligible templates, then guarantee that the new template is in the list.
		if len(step.TemplateIDs) > 0 {
			if templateID == "" {
				templateID = step.TemplateIDs[0]
			} else if !compare.Contains(step.TemplateIDs, templateID) {
				return derp.NewBadRequestError(location, "Cannot create new template of this kind", templateID)
			}
		}

		// Try to load the template for the new stream
		template, err := factory.Template().Load(templateID)

		if err != nil {
			return derp.Wrap(err, location, "Cannot find template", templateID)
		}

		// Verify that the new stream can be put into the parent stream
		if !template.CanBeContainedBy(renderer.templateRole()) {
			return derp.NewInternalError(location, "Template cannot be placed here", templateID)
		}

		// Create the new child stream
		child := model.NewStream()
		child.ParentID = renderer.objectID()
		child.TopLevelID = renderer.TopLevelID()
		// ParentIDs: child.ParentIDs = append(parent.ParentIDs, parent.StreamID)
		child.TemplateID = templateID

		// TODO: MEDIUM: sort order?
		// TODO: MEDIUM: presets defined by templates?

		return finalizeAddStream(factory, context, &child, template, step.WithChild)
	*/
	return nil
}

func (step StepAddChildEmbed) getBestTemplate(templates []form.LookupCode, templateID string) string {

	if len(templates) == 0 {
		return ""
	}

	for _, template := range templates {
		if template.Value == templateID {
			return templateID
		}
	}

	return templates[0].Value

}

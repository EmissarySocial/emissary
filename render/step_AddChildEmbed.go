package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
)

// StepAddChildEmbed is an action that can add new sub-streams to the domain.
type StepAddChildEmbed struct {
	TemplateIDs []string // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
}

func (step StepAddChildEmbed) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddChildEmbed.Get"

	templates, selectedTemplateID, childRenderer, err := step.common(renderer)

	if err != nil {
		return derp.Wrap(err, location, "Error getting common data")
	}

	iconService := renderer.factory().Icons()
	b := html.New()

	path := renderer.context().Request().URL.Path
	path = replaceActionID(path, renderer.ActionID())

	// Build the HTML for the "embed" widget
	b.Div().Data("hx-target", "this").Data("hx-swap", "outerHTML").EndBracket()

	if len(templates) > 0 {
		b.Div().Class("text-lg")
		for _, template := range templates {

			b.A("").Data("hx-get", path+"?templateId="+template.Value).EndBracket()

			if selectedTemplateID == template.Value {
				iconService.Write(template.Icon+"-fill", b)
			} else {
				iconService.Write(template.Icon, b)
			}

			b.Close()
			b.WriteString("&nbsp;")
		}
		b.Close()
	}

	// If there is a child renderer, then render it here
	if childRenderer != nil {

		widgetHTML, err := childRenderer.Render()

		if err != nil {
			return derp.Wrap(err, location, "Error rendering new child stream")
		}

		b.WriteString(string(widgetHTML))
	}

	// Close the container
	b.Close()

	// Write the whole widget back to the outpub buffer
	buffer.Write(b.Bytes())
	return nil
}

func (step StepAddChildEmbed) UseGlobalWrapper() bool {
	return true
}

func (step StepAddChildEmbed) Post(renderer Renderer) error {

	const location = "render.StepAddChildEmbed.Post"

	// Get pre-requisites
	_, _, childRenderer, err := step.common(renderer)

	if err != nil {
		return derp.Wrap(err, location, "Error getting common data")
	}

	// If there is no selected Template, then do nothing.
	if childRenderer == nil {
		return nil
	}

	// Get the "create" action from the template
	action := childRenderer.template().Action("create")

	if action == nil {
		return derp.NewInternalError(location, "No 'create' action found in template", childRenderer.template().TemplateID)
	}

	// Execute the "CREATE" pipeline to save the new stream.
	if err := Pipeline(action.Steps).Execute(renderer.factory(), childRenderer, renderer.context().Response(), ActionMethodPost); err != nil {
		return derp.Wrap(err, location, "Error executing pipeline")
	}

	// Done.  Return a regular widget.
	return nil
}

func (step StepAddChildEmbed) common(renderer Renderer) ([]form.LookupCode, string, Renderer, error) {

	const location = "render.StepAddChildEmbed.common"

	// Get prerequisites
	factory := renderer.factory()
	context := renderer.context()
	templateService := factory.Template()

	// Query all eligible templates
	templates := templateService.ListByContainerLimited(renderer.templateRole(), step.TemplateIDs)

	if len(templates) == 0 {
		return nil, "", nil, derp.NewBadRequestError(location, "No child templates available for this stream", renderer.templateRole())
	}

	// Find the "selected" template
	templateID := step.getBestTemplate(templates, context.QueryParam("templateId"))

	// If no valid template is selected, then do not render a child widget.
	if templateID == "" {
		return templates, templateID, nil, nil
	}

	streamService := factory.Stream()

	// Create a new child stream
	child, template, err := streamService.New(renderer.TopLevelID(), renderer.objectID(), templateID)

	if err != nil {
		return nil, "", nil, derp.Wrap(err, location, "Error creating new child stream")
	}

	// Create a new child renderer
	childRenderer, err := NewStream(factory, context, template, &child, "create")

	if err != nil {
		return nil, "", nil, derp.Wrap(err, location, "Error creating new child stream renderer")
	}

	return templates, templateID, childRenderer, nil

}

func (step StepAddChildEmbed) getBestTemplate(templates []form.LookupCode, templateID string) string {

	for _, template := range templates {
		if template.Value == templateID {
			return templateID
		}
	}

	return ""
}

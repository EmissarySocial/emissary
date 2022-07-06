package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/compare"
	"github.com/labstack/echo/v4"
)

// StepAddChildStream is an action that can add new sub-streams to the domain.
type StepAddChildStream struct {
	Title       string
	TemplateIDs []string    // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	View        string      // If present, use this HTML template as a custom "create" page.  If missing, a default modal pop-up is used.
	WithChild   []step.Step // List of steps to take on the newly created child record on POST.
}

func (step StepAddChildStream) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddChildStream.Get"

	// This can only be used on a Stream Renderer
	streamRenderer := renderer.(*Stream)

	// If a view has been specified, then use it to render a "create" page
	if step.View != "" {

		if err := renderer.executeTemplate(buffer, step.View, renderer); err != nil {
			return derp.Wrap(err, location, "Error executing template")
		}

		return nil
	}

	// Fall through to displaying the default modal
	modalAddStream(renderer.context().Response(), renderer.factory().Template(), step.Title, buffer, streamRenderer.URL(), streamRenderer.TemplateID(), step.TemplateIDs)

	return nil
}

func (step StepAddChildStream) UseGlobalWrapper() bool {
	return true
}

func (step StepAddChildStream) Post(renderer Renderer) error {

	const location = "render.StepAddChildStream.Post"

	// Collect prerequisites
	streamRenderer := renderer.(*Stream)
	context := streamRenderer.context()
	parent := streamRenderer.stream
	templateID := streamRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.TemplateIDs) > 0 {
		if templateID == "" {
			templateID = step.TemplateIDs[0]
		} else if !compare.Contains(step.TemplateIDs, templateID) {
			return derp.NewBadRequestError(location, "Cannot create new template of this kind", templateID)
		}
	}

	// Try to load the template for the new stream
	template, err := renderer.factory().Template().Load(templateID)

	if err != nil {
		return derp.Wrap(err, location, "Cannot find template")
	}

	// Verify that the new stream belongs in the parent stream
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.NewInternalError(location, "Template cannot be placed at top level", templateID)
	}

	// Create the new child stream
	child := model.NewStream()
	child.ParentID = parent.StreamID
	child.ParentIDs = append(parent.ParentIDs, parent.StreamID)
	child.TemplateID = templateID

	// TODO: sort order?
	// TODO: presets defined by templates?

	return finalizeAddStream(renderer.factory(), context, &child, template, step.WithChild)
}

// modalAddStream renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func modalAddStream(response *echo.Response, templateService *service.Template, title string, buffer io.Writer, url string, parentTemplateID string, allowedTemplateIDs []string) {

	templates := templateService.ListByContainerLimited(parentTemplateID, allowedTemplateIDs)

	b := html.New()

	b.H2().InnerHTML(title).Close()
	b.Table().Class("table space-below")

	for _, template := range templates {
		b.TR().Role("link").Data("hx-post", url+"?templateId="+template.Value)
		{
			b.TD()
			b.I(template.Icon + " fa-3x gray").Close()
			b.Close()

			b.TD().Style("width:100%")
			b.Div().Class("big", "bold").InnerHTML(template.Label).Close()
			b.Div().Class("small", "gray").InnerHTML(template.Description).Close()
			b.Close()
		}
		b.Close()
	}

	b.CloseAll()

	result := WrapModalWithCloseButton(response, b.String())

	io.WriteString(buffer, result)
}

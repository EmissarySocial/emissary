package render

import (
	"html/template"

	"github.com/benpate/derp"
)

// Domain contains a domain configuration and knows how to render it into HTML
type Domain struct {
	templateService TemplateService
	renderer        Renderer
	view            string
}

// NewDomain returns a fully initialized DomaainWrapper
func NewDomain(templateService TemplateService, renderer Renderer, view string) Domain {

	return Domain{
		templateService: templateService,
		renderer:        renderer,
		view:            view,
	}
}

// Render generates the HTML output of the chrome, or wrapper for this domain
func (w *Domain) Render() (string, error) {

	// Try to load the template from the database
	template, err := w.templateService.Load("domain")

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Domain.Render", "Unable to load Domain Template")
	}

	// Locate / Authenticate the view to use
	view, err := template.View("default", w.view)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Domain.Render", "Unrecognized view", w.view)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Domain.Render", "Error rendering view")
	}

	// TODO: Add caching here??

	// Success!
	return result, nil
}

// StreamID provides the current StreamID being generated -- used by templates to render HTML
func (w *Domain) StreamID() string {
	return w.renderer.StreamID()
}

// Token provides the current URL token for the stream being generated -- used by templates to render HTML
func (w *Domain) Token() string {
	return w.renderer.Token()
}

// InnerHTML returns the HTML representation of the innerHTML content -- used by templates to render HTML
func (w *Domain) InnerHTML() template.HTML {
	innerHTML, err := w.renderer.Render()

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.render.Domain.InnerHTML", "Error rendering innerHTML"))
	}

	return template.HTML(innerHTML)
}

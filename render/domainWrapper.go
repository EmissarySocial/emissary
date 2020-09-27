package render

import (
	"html/template"

	"github.com/benpate/derp"
)

// DomainWrapper contains a domain configuration and knows how to render it into HTML
type DomainWrapper struct {
	templateService TemplateService
	renderer        Renderer
	view            string
}

// NewDomainWrapper returns a fully initialized DomaainWrapper
func NewDomainWrapper(templateService TemplateService, renderer Renderer, view string) DomainWrapper {

	return DomainWrapper{
		templateService: templateService,
		renderer:        renderer,
		view:            view,
	}
}

// Render generates the HTML output of the chrome, or wrapper for this domain
func (w *DomainWrapper) Render() (string, error) {

	// Try to load the template from the database
	template, err := w.templateService.Load("domain")

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unable to load Domain Template")
	}

	// Locate / Authenticate the view to use
	view, err := template.View("default", w.view)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unrecognized view", w.view)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Error rendering view")
	}

	// TODO: Add caching here...

	// Success!
	return result, nil
}

// Stream returns the StreamWrapper for the stream being generated -- used by templates to render HTML
func (w *DomainWrapper) Stream() StreamWrapper {
	return w.renderer.Stream()
}

// StreamID provides the current StreamID being generated -- used by templates to render HTML
func (w *DomainWrapper) StreamID() string {
	return w.renderer.Stream().StreamID()
}

// Token provides the current URL token for the stream being generated -- used by templates to render HTML
func (w *DomainWrapper) Token() string {
	return w.renderer.Stream().Token()
}

// View provides the name of the view being generated -- used by templates to render HTML
func (w *DomainWrapper) View() string {
	return w.renderer.Stream().view
}

// InnerHTML returns the HTML representation of the innerHTML content -- used by templates to render HTML
func (w *DomainWrapper) InnerHTML() template.HTML {
	innerHTML, err := w.renderer.Render()

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.render.DomainWrapper.InnerHTML", "Error rendering innerHTML"))
	}

	return template.HTML(innerHTML)
}

package render

import (
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// DomainWrapper contains a domain configuration and knows how to render it into HTML
type DomainWrapper struct {
	factory    *service.Factory
	stream     *StreamWrapper
	domainView string
	streamView string
	innerHTML  *string
}

// NewDomainWrapper returns a fully initialized DomaainWrapper
func NewDomainWrapper(factory *service.Factory, stream *StreamWrapper, domainView string, streamView string, innerHTML *string) *DomainWrapper {

	return &DomainWrapper{
		factory:    factory,
		stream:     stream,
		domainView: domainView,
		streamView: streamView,
		innerHTML:  innerHTML,
	}
}

// Render generates the HTML output of the chrome, or wrapper for this domain
func (w *DomainWrapper) Render() (*string, error) {

	templateService := w.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load("domain")

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unable to load Domain Template")
	}

	// Locate / Authenticate the view to use
	view, err := template.View("default", w.domainView)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unrecognized view", w.domainView)
	}

	// TODO: need to enforce permissions somewhere...

	// Try to generate the HTML response using the provided data
	result, err := view.Execute(w)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Error rendering view")
	}

	// TODO: Add caching here...

	// Success!
	return &result, nil
}

// StreamID provides the current StreamID being generated -- used by templates to render HTML
func (w *DomainWrapper) StreamID() string {
	return w.stream.StreamID()
}

// Token provides the current URL token for the stream being generated -- used by templates to render HTML
func (w *DomainWrapper) Token() string {
	return w.stream.Token()
}

// View provides the name of the view being generated -- used by templates to render HTML
func (w *DomainWrapper) View() string {
	return w.streamView
}

// InnerHTML returns the HTML representation of the innerHTML content -- used by templates to render HTML
func (w *DomainWrapper) InnerHTML() template.HTML {
	return template.HTML(*w.innerHTML)
}

// Stream returns the StreamWrapper for the stream being generated -- used by templates to render HTML
func (w *DomainWrapper) Stream() *StreamWrapper {
	return w.stream
}

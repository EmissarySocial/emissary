package render

import (
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

type DomainWrapper struct {
	factory   *service.Factory
	stream    *StreamWrapper
	innerHTML *string
}

func NewDomainWrapper(factory *service.Factory, stream *StreamWrapper, innerHTML *string) *DomainWrapper {

	return &DomainWrapper{factory: factory, stream: stream, innerHTML: innerHTML}
}

func (w *DomainWrapper) Render(viewName string) (*string, error) {

	templateService := w.factory.Template()

	// Try to load the template from the database
	template, err := templateService.Load("domain")

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unable to load Domain Template")
	}

	// Locate / Authenticate the view to use
	view, err := template.View("default", viewName)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.render.DomainWrapper.Render", "Unrecognized view", viewName)
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

func (w *DomainWrapper) InnerHTML() template.HTML {
	return template.HTML(*w.innerHTML)
}

func (w *DomainWrapper) Stream() *StreamWrapper {
	return w.stream
}

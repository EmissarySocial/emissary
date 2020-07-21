package templatesource

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"
)

// TemplateSource is any dataprovider that can read and write Templates.  The TemplateService can
// support multiple TemplateSource objects
type TemplateSource interface {

	// Register connects the TemplateSource to the TemplateService, so that the
	// TemplateSource can write/update Templates in the TemplateService as needed.
	Register(TemplateService)

	// List returns a list of the templates that this source can access
	List() ([]string, *derp.Error)

	// Load tries to locate a Template from the TemplateSource data
	Load(string) (model.Template, *derp.Error)
}

// TemplateService interface manages templates outside of this library.
type TemplateService interface {
	Cache(model.Template)
}

func Populate(service TemplateService, source TemplateSource) {

	spew.Dump("Populating")
	if list, err := source.List(); err == nil {
		spew.Dump(list)

		for _, token := range list {
			spew.Dump(token)

			if template, err := source.Load(token); err == nil {

				service.Cache(template)
				spew.Dump("Pushed: " + template.TemplateID)
			}
		}
	}
}

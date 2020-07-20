package templatesource

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// TemplateSource is any dataprovider that can read and write Templates.  The TemplateService can
// support multiple TemplateSource objects
type TemplateSource interface {

	// List returns a list of the templates that this source can access
	List() ([]string, *derp.Error)

	// Load tries to locate a Template from the TemplateSource data
	Load(string) (*model.Template, *derp.Error)

	// Save tries to locate a Template from the TemplateSource data
	Save(*model.Template, string) *derp.Error
}

// RealtimeTemplateSource is a sub-set of TemplateSource that can also push realtime Template updates
// back into the Template service.
type RealtimeTemplateSource interface {

	// RegisterRealtime links a TemplateSource to the Template service, and gives it
	// a way to push new objects into the service (for instance, watching a directory or mongodb collection)
	RegisterRealtime(TemplateService)
}

type TemplateService interface {
	Save(*model.Template, string) *derp.Error
}

package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// TemplateSource is any dataprovider that can read and write Templates.  The TemplateService can
// support multiple TemplateSource objects
type TemplateSource interface {

	// ID returns a unique identifier for this TemplateSource, so that templates in memory can be
	// linked to the correct source.
	ID() string

	// Load tries to locate a Template from the TemplateSource data
	Load(string) (model.Template, *derp.Error)
}

package service

import (
	"github.com/benpate/ghost/model"
)

// TemplateSource is any dataprovider that can read and write Templates.  The TemplateService can
// support multiple TemplateSource objects
type TemplateSource interface {

	// List returns a list of the templates that this source can access
	List() ([]string, error)

	// Load tries to locate a Template from the TemplateSource data
	Load(string) (*model.Template, error)

	// Watch passes realtime updates to templates back through to the provided channel
	Watch(chan model.Template) error
}

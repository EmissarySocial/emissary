package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// CollectionTemplate is the database collection where Templates are stored
const CollectionTemplate = "Template"

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	Sources   []TemplateSource
	Templates map[string]*model.Template
}

// New creates a newly initialized Key that is ready to use
func (service Template) New() *model.Template {
	return &model.Template{}
}

// List returns an iterator containing all of the Templates who match the provided criteria
func (service Template) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return nil, derp.New(500, "ghost.service.Template.List", "Unimplemented")
}

// Load retrieves an Template from the database
func (service Template) Load(templateID string) (*model.Template, *derp.Error) {

	// Look in the local cache first
	if template, ok := service.Templates[templateID]; ok {
		return template, nil
	}

	// Otherwise, search all sources for the Template.
	for index := range service.Sources {
		if template, err := service.Sources[index].Load(templateID); err != nil {
			service.Templates[templateID] = &template
			return &template, nil
		}
	}

	return nil, derp.New(404, "ghost.sevice.Template.Load", "Could not load Template", templateID)
}

// Save adds/updates an Template in the database
func (service Template) Save(template *model.Template, note string) *derp.Error {
	service.Templates[template.TemplateID] = template
	// TODO: should this also persist to TemplateSources???

	return nil
}

// Delete removes an Template from the database (virtual delete)
func (service Template) Delete(template *model.Template, note string) *derp.Error {
	return derp.New(500, "ghost.service.Template.Delete", "Unimplemented")
}

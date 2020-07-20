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
	Templates map[string]model.Template
}

// Startup loads all templates from all available sources.
func (service *Template) Startup(sources []TemplateSource) []*derp.Error {

	var errors []*derp.Error

	service.Sources = sources

	// Iterate through every source
	for _, source := range service.Sources {

		list, err := source.List()

		if err != nil {
			errors = append(errors, err)
			continue
		}

		// Iterate through every template
		for _, name := range list {

			template, err := source.Load(name)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			// Save the template in memory.
			service.Cache(template)
		}
	}

	return errors
}

// List returns an iterator containing all of the Templates who match the provided criteria
func (service Template) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return nil, derp.New(500, "ghost.service.Template.List", "Unimplemented")
}

// Load retrieves an Template from the database
func (service Template) Load(templateID string) (model.Template, *derp.Error) {

	// Look in the local cache first
	if template, ok := service.Templates[templateID]; ok {
		return template, nil
	}

	// Otherwise, search all sources for the Template.
	for index := range service.Sources {
		if template, err := service.Sources[index].Load(templateID); err != nil {
			service.Templates[templateID] = template
			return template, nil
		}
	}

	return model.Template{}, derp.New(404, "ghost.sevice.Template.Load", "Could not load Template", templateID)
}

// Cache adds/updates an Template in the memory cache
func (service Template) Cache(template model.Template) {
	service.Templates[template.TemplateID] = template
}

// Delete removes an Template from the database (virtual delete)
func (service Template) Delete(template model.Template, note string) *derp.Error {
	delete(service.Templates, template.TemplateID)
	return nil
}

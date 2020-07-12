package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionTemplate is the database collection where Templates are stored
const CollectionTemplate = "Template"

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Key that is ready to use
func (service Template) New() *model.Template {
	return &model.Template{
		TemplateID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Templates who match the provided criteria
func (service Template) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.session.List(CollectionTemplate, criteria, options...)
}

// Load retrieves an Template from the database
func (service Template) Load(criteria expression.Expression) (*model.Template, error) {

	template := service.New()

	if err := service.session.Load(CollectionTemplate, criteria, template); err != nil {
		return nil, derp.Wrap(err, "service.Template", "Error loading Template", criteria)
	}

	return template, nil
}

// Save adds/updates an Template in the database
func (service Template) Save(template *model.Template, note string) error {

	if err := service.session.Save(CollectionTemplate, template, note); err != nil {
		return derp.Wrap(err, "service.Template", "Error saving Template", template, note)
	}

	return nil
}

// Delete removes an Template from the database (virtual delete)
func (service Template) Delete(template *model.Template, note string) error {

	if err := service.session.Delete(CollectionTemplate, template, note); err != nil {
		return derp.Wrap(err, "service.Template", "Error deleting Template", template, note)
	}

	return nil
}

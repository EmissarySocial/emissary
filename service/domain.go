package service

import (
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/whisperverse/whisperverse/model"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection data.Collection
	funcMap    template.FuncMap
}

// NewDomain returns a fully initialized Domain service
func NewDomain(collection data.Collection, funcMap template.FuncMap) Domain {
	return Domain{
		collection: collection,
		funcMap:    funcMap,
	}
}

// Load retrieves an Domain from the database
func (service *Domain) Load(domain *model.Domain) error {

	if err := service.collection.Load(exp.All(), domain); err != nil {
		return derp.Wrap(err, "service.Domain.Load", "Error loading Domain")
	}

	return nil
}

func (service *Domain) Save(domain *model.Domain, note string) error {

	if err := service.collection.Save(domain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error saving Domain")
	}

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	result := model.NewDomain()
	return &result
}

func (service *Domain) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.New(derp.CodeBadRequestError, "service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectLoad(_ exp.Expression) (data.Object, error) {
	result := model.NewDomain()
	err := service.Load(&result)
	return &result, err
}

func (service *Domain) ObjectSave(object data.Object, note string) error {
	return service.Save(object.(*model.Domain), note)
}

func (service *Domain) ObjectDelete(object data.Object, note string) error {
	return derp.New(derp.CodeBadRequestError, "service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) Debug() datatype.Map {
	return datatype.Map{
		"service": "Domain",
	}
}

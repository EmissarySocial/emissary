package service

import (
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
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
		return derp.Wrap(err, "ghost.service.Domain", "Error loading Domain")
	}

	return nil
}

func (service *Domain) Save(domain *model.Domain, comment string) error {
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
	return nil, derp.New(derp.CodeBadRequestError, "ghost.service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectLoad(_ exp.Expression) (data.Object, error) {
	result := model.NewDomain()
	err := service.Load(&result)
	return &result, err
}

func (service *Domain) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.Domain), comment)
}

func (service *Domain) ObjectDelete(object data.Object, comment string) error {
	return derp.New(derp.CodeBadRequestError, "ghost.service.Domain.ObjectDelete", "Unsupported")
}

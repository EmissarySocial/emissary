package service

import (
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection data.Collection
	funcMap    template.FuncMap
	template   *template.Template
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

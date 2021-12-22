package render

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
)

// ModelService interface wraps the generic Object-* functions that standard services provide
type ModelService interface {
	ObjectNew() data.Object
	ObjectList(exp.Expression, ...option.Option) (data.Iterator, error)
	ObjectLoad(exp.Expression) (data.Object, error)
	ObjectSave(data.Object, string) error
	ObjectDelete(data.Object, string) error
}

// getModelService wraps that factory methods for various standard services.
func getModelService(factory Factory, modelServiceType string) ModelService {
	switch modelServiceType {
	case "user":
		return factory.User()

	case "group":
		return factory.Group()

	case "stream":
		return factory.Stream()

	default:
		return factory.Stream()
	}
}

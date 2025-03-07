package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModelService interface wraps the generic Object-* functions that standard services provide
type ModelService interface {
	ObjectType() string
	ObjectID(data.Object) primitive.ObjectID
	ObjectNew() data.Object
	ObjectQuery(any, exp.Expression, ...option.Option) error
	ObjectLoad(exp.Expression) (data.Object, error)
	ObjectSave(data.Object, string) error
	ObjectDelete(data.Object, string) error
	ObjectUserCan(data.Object, model.Authorization, string) error
	Count(exp.Expression) (int64, error)
	Schema() schema.Schema
}

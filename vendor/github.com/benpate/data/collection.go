package data

import (
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
)

// Collection represents a single database collection (or table) that is opened to support a single transactional request, and then closed
// when this transaction is complete
type Collection interface {
	List(filter exp.Expression, options ...option.Option) (Iterator, error)
	Load(filter exp.Expression, target Object) error
	Save(object Object, note string) error
	Delete(object Object, note string) error
	HardDelete(object Object) error
}

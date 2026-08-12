package model

import (
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

// TagList satisfies schema.ArrayGetterSetter through its POINTER (SetIndex has a pointer receiver).
// schema.validate_Array requires this interface for any value used as an Array property, and this
// assertion moves a failure there from runtime validation to compile time -- the same guard rosetta
// uses for its own sliceof types.  It holds because TagList is an alias for sliceof.Object[Tag]
// rather than a defined type; see tagList.go.
var _ schema.ArrayGetterSetter = (*sliceof.Object[Tag])(nil)

// TagListSchema describes a document's `tags` array.  There are no getter/setter methods to
// declare here: TagList inherits them from sliceof.Object, and the per-Tag accessors live in
// tag_accessors.go.
func TagListSchema() schema.Element {
	return schema.Array{
		Items: TagSchema(),
	}
}

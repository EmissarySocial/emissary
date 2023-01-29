package activitystreams

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

type Context struct {
	ID         string
	Extensions mapof.String
}

func (context *Context) UnmarshalJSON(data []byte) error {
	return derp.NewInternalError("activitystreams.Context.MarshalJSON", "Not implemented", nil)
}

func (context Context) MarshalJSON() ([]byte, error) {
	return []byte{}, derp.NewInternalError("activitystreams.Context.MarshalJSON", "Not implemented", nil)
}

package action

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// UpdateContent manages the content.Content in a stream.
type UpdateContent struct {
	CommonInfo
}

func (action UpdateContent) Get(ctx steranko.Context, stream *model.Stream) error {
	return nil
}
func (action UpdateContent) Post(ctx steranko.Context, stream *model.Stream) error {
	return nil
}

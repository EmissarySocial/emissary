package render

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// PublishContent manages the content.Content in a stream.
type PublishContent struct {
	factory Factory
	model.ActionConfig
}

func NewAction_PublishContent(factory Factory, config model.ActionConfig) PublishContent {
	return PublishContent{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action PublishContent) Get(renderer Renderer) (string, error) {
	return "", nil
}
func (action PublishContent) Post(ctx *steranko.Context, stream *model.Stream) error {
	return nil
}

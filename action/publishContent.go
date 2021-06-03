package action

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

// PublishContent manages the content.Content in a stream.
type PublishContent struct {
	model.ActionConfig
	streamService *service.Stream
}

func NewAction_PublishContent(config model.ActionConfig, streamService *service.Stream) PublishContent {
	return PublishContent{
		ActionConfig:  config,
		streamService: streamService,
	}
}

func (action PublishContent) Get(ctx steranko.Context, stream *model.Stream) error {
	return nil
}
func (action PublishContent) Post(ctx steranko.Context, stream *model.Stream) error {
	return nil
}

package action

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

// PublishContent manages the content.Content in a stream.
type PublishContent struct {
	config        model.ActionConfig
	streamService *service.Stream
}

func NewAction_PublishContent(config model.ActionConfig, streamService *service.Stream) PublishContent {
	return PublishContent{
		config:        config,
		streamService: streamService,
	}
}

func (action *PublishContent) Get(ctx steranko.Context, stream *model.Stream) error {
	return nil
}
func (action *PublishContent) Post(ctx steranko.Context, stream *model.Stream) error {
	return nil
}

// Config returns the configuration information for this action
func (action *PublishContent) Config() model.ActionConfig {
	return action.config
}

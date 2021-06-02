package action

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

// UpdateContent manages the content.Content in a stream.
type UpdateContent struct {
	config        model.ActionConfig
	streamService *service.Stream
}

func NewAction_UpdateContent(config model.ActionConfig, streamService *service.Stream) UpdateContent {
	return UpdateContent{
		config:        config,
		streamService: streamService,
	}
}

func (action *UpdateContent) Get(ctx steranko.Context, stream *model.Stream) error {
	return nil
}
func (action *UpdateContent) Post(ctx steranko.Context, stream *model.Stream) error {
	return nil
}

// Config returns the configuration information for this action
func (action *UpdateContent) Config() model.ActionConfig {
	return action.config
}

package render

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// UpdateDraft manages the content.Content in a stream.
type UpdateDraft struct {
	factory Factory
	model.ActionConfig
}

func NewAction_UpdateDraft(factory Factory, config model.ActionConfig) UpdateDraft {
	return UpdateDraft{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action UpdateDraft) Get(renderer Renderer) (string, error) {
	return "", nil
}
func (action UpdateDraft) Post(ctx steranko.Context, stream *model.Stream) error {
	return nil
}

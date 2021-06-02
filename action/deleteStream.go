package action

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

type DeleteStream struct {
	config        model.ActionConfig
	streamService *service.Stream
}

func NewAction_DeleteStream(config model.ActionConfig, streamService *service.Stream) DeleteStream {
	return DeleteStream{
		config:        config,
		streamService: streamService,
	}
}

func (action *DeleteStream) Get(ctx steranko.Context, stream *model.Stream) error {
	return derp.New(derp.CodeBadRequestError, "ghost.model.action.DeleteStream.Get", "Unsupported Method")
}

func (action *DeleteStream) Post(ctx steranko.Context, stream *model.Stream) error {

	var parent model.Stream

	if err := action.streamService.LoadParent(stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.model.action.DeleteStream.Post", "Error loading parent stream")
	}

	if err := action.streamService.Delete(stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.model.action.DeleteStream.Post", "Error deleting stream")
	}

	ctx.Response().Header().Add("hx-redirect", "/"+parent.Token)
	return ctx.NoContent(http.StatusNoContent)
}

// Config returns the configuration information for this action
func (action *DeleteStream) Config() model.ActionConfig {
	return action.config
}

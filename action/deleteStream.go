package action

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

type DeleteStream struct {
	CommonInfo
	streamService *service.Stream
}

func NewAction_DeleteStream(config *model.ActionConfig, streamService *service.Stream) DeleteStream {
	return DeleteStream{
		CommonInfo:    NewCommonInfo(config),
		streamService: streamService,
	}
}

func (action DeleteStream) Get(ctx echo.Context, stream *model.Stream) error {
	return derp.New(derp.CodeBadRequestError, "ghost.model.action.DeleteStream.Get", "Unsupported Method")
}

func (action DeleteStream) Post(ctx echo.Context, stream *model.Stream) error {

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

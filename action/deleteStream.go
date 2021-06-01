package action

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/labstack/echo/v4"
)

type DeleteStream struct {
	Info
}

func (action DeleteStream) Get(ctx echo.Context, factory *domain.Factory, stream *model.Stream) error {
	return derp.New(derp.CodeBadRequestError, "ghost.model.action.DeleteStream.Get", "Unsupported Method")
}

func (action DeleteStream) Post(ctx echo.Context, factory *domain.Factory, stream *model.Stream) error {

	var parent model.Stream

	streamService := factory.Stream()

	if err := streamService.LoadParent(stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.model.action.DeleteStream.Post", "Error loading parent stream")
	}

	if err := streamService.Delete(stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.model.action.DeleteStream.Post", "Error deleting stream")
	}

	ctx.Response().Header().Add("hx-redirect", "/"+parent.Token)
	return ctx.NoContent(http.StatusNoContent)
}

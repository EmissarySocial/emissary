package action

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

type CreateChild struct {
	streamService *service.Stream
	childStateID  string
	templateID    string
	Info
}

func (action CreateChild) Get(request domain.HTTPRequest, stream *model.Stream) (string, error) {
	return "", nil
}

func (action CreateChild) Put(request domain.HTTPRequest, stream *model.Stream) (string, error) {

	child := action.streamService.New()

	child.ParentID = stream.StreamID
	child.AuthorID = request.Authorization().UserID
	child.StateID = action.childStateID

	if err := action.streamService.Save(&child, "created"); err != nil {
		return "", derp.Wrap(err, "ghost.action.CreateChild.Post", "Error saving child")
	}

	return "", nil
}

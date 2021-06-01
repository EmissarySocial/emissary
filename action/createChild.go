package action

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type CreateChild struct {
	ChildStateID string
	TemplateID   string
	Info
}

func (action CreateChild) Get(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) (string, error) {
	return "", nil
}

func (action CreateChild) Put(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) (string, error) {

	streamService := factory.Stream()
	child := streamService.New()

	authorization := getAuthorization(ctx)

	child.ParentID = stream.StreamID
	child.AuthorID = authorization.UserID
	child.StateID = action.ChildStateID

	if err := streamService.Save(&child, "created"); err != nil {
		return "", derp.Wrap(err, "ghost.action.CreateChild.Post", "Error saving child")
	}

	return "", nil
}

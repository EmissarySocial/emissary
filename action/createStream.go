package action

import (
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type CreateStream struct {
	Info
}

func Get(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) error {
	return nil
}

func Post(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) error {
	return nil
}

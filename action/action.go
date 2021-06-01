package action

import (
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// Action configures an individual action function that will be executed when a stream transitions from one state to another.
type Action interface {
	Get(steranko.Context, *domain.Factory, *model.Stream) error
	Post(steranko.Context, *domain.Factory, *model.Stream) error
	UserCan(*model.Stream, *model.Authorization) bool
}

func Parse(config model.ActionConfig) Action {

	switch config.Method {

	}

	return nil
}

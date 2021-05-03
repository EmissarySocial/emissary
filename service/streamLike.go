package service

import (
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
)

// StreamLike interface wraps services that can perform basic operations on model.Streams
type StreamLike interface {
	Load(exp.Expression) (*model.Stream, error)
	Save(*model.Stream, string) error
	Delete(*model.Stream, string) error
	ByToken(string) (*model.Stream, error)
}

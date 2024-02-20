package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/pub"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory *domain.Factory
	stream  *model.Stream
	actor   *model.Actor
}

// streamRouter defines the package-level router for stream/ActivityPub requests
var streamRouter pub.Router[Context] = pub.NewRouter[Context]()

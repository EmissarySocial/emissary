package service

import "github.com/benpate/hannibal/streams"

// actorLoader resolves a Fediverse address (a webfinger handle or a profile URL) to its canonical
// Actor document. *ActivityStream satisfies it; depending on this one method keeps services testable.
type actorLoader interface {
	GetActor(string) (streams.Document, error)
}

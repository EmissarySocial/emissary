package consumer

import (
	"iter"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/turbine/queue"
)

type ServerFactory interface {
	RangeDomains() iter.Seq[*domain.Factory]
	ByHostname(hostname string) (*domain.Factory, error)
	Queue() *queue.Queue
}

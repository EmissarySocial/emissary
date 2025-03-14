package consumer

import (
	"iter"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/turbine/queue"
)

type ServerFactory interface {
	RangeDomains() iter.Seq[*domain.Factory]
	ByDomainName(domain string) (*domain.Factory, error)
	Queue() *queue.Queue
}

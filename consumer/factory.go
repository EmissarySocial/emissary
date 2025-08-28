package consumer

import (
	"iter"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServerFactory interface {
	RangeDomains() iter.Seq[*service.Factory]
	ByHostname(hostname string) (*service.Factory, error)
	Queue() *queue.Queue
	CommonDatabase() *mongo.Database
}

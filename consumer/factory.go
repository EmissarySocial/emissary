package consumer

import (
	"iter"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServerFactory interface {
	RangeDomains() iter.Seq[*domain.Factory]
	ByHostname(hostname string) (*domain.Factory, error)
	Queue() *queue.Queue
	CommonDatabase() *mongo.Database
	ReadSession() (data.Session, error)
	WriteSession() (data.Session, error)
}

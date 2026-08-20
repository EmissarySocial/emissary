package consumer

import (
	"iter"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/mongo"
)

// ServerFactory is the slice of the server Factory that queue consumers depend on
type ServerFactory interface {
	RangeDomains() iter.Seq[*service.Factory]
	ByHostname(hostname string) (*service.Factory, error)
	Queue() *queue.Queue
	CommonDatabase() *mongo.Database

	// AllowPrivateIPs reports whether outbound ActivityPub delivery may connect to
	// non-public (private/loopback) addresses. FALSE in production; enabled only for
	// local/dev federation between machines on a private network.
	AllowPrivateIPs() bool
}

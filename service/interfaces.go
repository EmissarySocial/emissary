package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ServerFactory is the domain Factory's window onto the server that owns it: the domain registry,
// server-level services, and the server-level resources (queue, common database) that a config
// reload can rebuild and must therefore be read through this interface on every use.
type ServerFactory interface {
	ByHostname(hostname string) (*Factory, error)
	Email() *ServerEmail
	ClientIP(request *http.Request) string
	DigitalDome() *dome.Dome

	// Queue returns the CURRENT task queue.  Domain factories must read it through this getter on
	// every use -- never capture the pointer -- because a config reload can rebuild the queue,
	// and a captured pointer then publishes tasks into a stopped queue that silently drops them.
	Queue() *queue.Queue

	// CommonDatabase returns the CURRENT connection to the shared (ActivityPub Cache) database,
	// or nil while disconnected (FACTORY-MODES D7).  Domain factories must read it through this
	// getter on every use -- never capture the value -- because a config reload can reconnect the
	// database, and a captured handle then fails every call with "client is disconnected".
	CommonDatabase() *mongo.Database
}

// DomainFactory is the narrow slice of a domain Factory that cross-domain tools (like the
// ActivityPub sender) need: identity, key access, and database sessions.
type DomainFactory interface {
	EncryptionKey() *EncryptionKey
	Hostname() string
	Locator() *Locator
	Session(time.Duration) (data.Session, context.CancelFunc, error)
}

// TemplateLike is anything that can execute itself against a data value and write the result --
// satisfied by both text/template and html/template Templates.
type TemplateLike interface {
	Execute(writer io.Writer, data any) error
}

// MerchantAccountAdapter abstracts one payment provider (Stripe, PayPal, ...) behind the
// MerchantAccount service: signup, API-key refresh, checkout, and webhook parsing.
type MerchantAccountAdapter interface {
	GetSignupURL(*model.Connection) (string, error)
	RefreshAPIKeys() error
	GetCheckoutURL() (string, error)
	ParseCheckoutResponse(url.Values) (model.Privilege, error)
	ParseCheckoutWebhook(http.Header, []byte) error
	SubscriptionCancelURL(string) (string, error)
}

// Exportable is a service that can export records
type Exportable interface {
	ExportCollection(data.Session, primitive.ObjectID) ([]model.IDOnly, error)
	ExportDocument(data.Session, primitive.ObjectID, primitive.ObjectID) (string, error)
}

// Importable is a service that can import records
type Importable interface {
	Import(data.Session, *model.Import, *model.ImportItem, *model.User, []byte) error
	UndoImport(data.Session, *model.ImportItem) error
}

// ImportableLocator is a function that can locate Importable services
type ImportableLocator func(string) (Importable, error)

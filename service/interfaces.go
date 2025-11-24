package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServerFactory interface {
	ByHostname(hostname string) (*Factory, error)
}

type DomainFactory interface {
	EncryptionKey() *EncryptionKey
	Hostname() string
	Locator() *Locator
	Session(time.Duration) (data.Session, context.CancelFunc, error)
}

type TemplateLike interface {
	Execute(writer io.Writer, data any) error
}

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
	Import(data.Session, *model.ImportItem, *model.User, []byte) error
	UndoImport(data.Session, *model.ImportItem) error
}

// ImportableLocator is a function that can locate Importable services
type ImportableLocator func(string) (Importable, error)

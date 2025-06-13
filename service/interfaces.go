package service

import (
	"io"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthorSetter interface {
	SetAuthor(object data.Object, authorID primitive.ObjectID) error
}

type TemplateLike interface {
	Execute(writer io.Writer, data any) error
}

type MerchantAccountAdapter interface {
	RefreshAPIKeys() error
	GetCheckoutURL() (string, error)
	ParseCheckoutResponse(url.Values) (model.Privilege, error)
	ParseCheckoutWebhook(http.Header, []byte) error
	SubscriptionCancelURL(string) (string, error)
}

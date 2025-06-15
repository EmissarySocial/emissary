package service

import (
	"io"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
)

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

package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

func (service *MerchantAccount) stripe_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {
	return derp.NewInternalError("service.MerchantAccount.stripe_refreshMerchantAccount", "Not Implemented")
}

func (service *MerchantAccount) stripe_getSubscriptions(merchantAccount *model.MerchantAccount) ([]form.LookupCode, error) {
	return nil, derp.NewInternalError("service.MerchantAccount.paypal_getSubscriptions", "Not Implemented")
}

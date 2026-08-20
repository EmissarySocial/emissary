package service

import "github.com/EmissarySocial/emissary/model"

// stripeConnect_Connect refreshes a Stripe Connect account. Connect accounts use permanent keys, so there is nothing to do.
func (service *MerchantAccount) stripeConnect_Connect(_ *model.MerchantAccount) error {
	return nil
}

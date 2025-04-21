package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
)

func (service *MerchantAccount) stripe_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {
	return nil
}

func (service *MerchantAccount) stripe_getSubscriptions(merchantAccount *model.MerchantAccount) ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.paypal_getSubscriptions"

	endpoint := "https://api.stripe.com/v1/products"
	txnResult := mapof.NewAny()

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving API keys")
	}

	txn := remote.Get(endpoint).
		ContentType("application/json").
		With(options.BearerAuth(restrictedKey)).
		Result(&txnResult)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error connecting to PayPal API")
	}

	spew.Dump(txnResult)
	products := txnResult.GetSliceOfAny("data")
	result := make([]form.LookupCode, len(products))

	for index, product := range products {
		productMap := mapof.Any(convert.MapOfAny(product))

		result[index] = form.LookupCode{
			Value:       productMap.GetString("id"),
			Label:       productMap.GetString("name"),
			Description: productMap.GetString("description"),
		}
	}

	return result, nil
}

func (service *MerchantAccount) stripe_getRestrictedKey(merchantAccount *model.MerchantAccount) (string, error) {

	const location = "service.MerchantAccount.stripe_getRestrictedKey"
	var propertyName string

	if merchantAccount.LiveMode {
		propertyName = "restrictedKey_live"
	} else {
		propertyName = "restrictedKey_test"
	}

	apiKeys, err := service.getAPIKeys(merchantAccount, propertyName)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving API keys")
	}

	return apiKeys.GetString(propertyName), nil
}

package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestMerchantAccountSchema(t *testing.T) {

	merchantAccount := NewMerchantAccount()
	s := schema.New(MerchantAccountSchema())

	table := []tableTestItem{
		{"merchantAccountId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"type", ConnectionProviderStripeConnect, nil},
		{"name", "My Shop", nil},
		{"description", "Storefront merchant account", nil},
		{"liveMode", true, nil},
		// note: "vault" values are obscured on read-back by design, so they are not round-trippable here.
	}

	tableTest_Schema(t, &s, &merchantAccount, table)
}

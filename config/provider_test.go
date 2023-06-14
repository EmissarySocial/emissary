package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestProviderSchema(t *testing.T) {

	d := NewProvider("GIPHY")
	s := schema.New(ProviderSchema())

	table := []tableTestItem{
		{"providerId", "GIPHY", nil},
		{"clientId", "CLIENT_ID", nil},
		{"clientSecret", "CLIENT_SECRET", nil},
	}

	tableTest_Schema(t, &s, &d, table)
}

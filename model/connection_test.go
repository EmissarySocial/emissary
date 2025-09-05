package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestConnection(t *testing.T) {

	origin := NewConnection()

	s := schema.New(ConnectionSchema())

	table := []tableTestItem{
		{"connectionId", "123456781234567812345678", nil},
		{"providerId", "GIPHY", nil},
		{"type", "USER-PAYMENT", nil},
		{"data.random", "Any Value", nil},
		{"data.liveMode", "true", nil},
		{"active", "true", true},
	}

	tableTest_Schema(t, &s, &origin, table)
}

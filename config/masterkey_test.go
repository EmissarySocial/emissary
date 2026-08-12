package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMasterKey(t *testing.T) {

	table := []struct {
		name      string
		masterKey string
		isValid   bool
	}{
		{"valid lowercase", "6dcc2f456c269b6cbb2012ad434981c8a868ee8a7ed1be8f3a75cec449d1b31f", true},
		{"valid uppercase", "6DCC2F456C269B6CBB2012AD434981C8A868EE8A7ED1BE8F3A75CEC449D1B31F", true},
		{"empty is a configuration error, not a valid key", "", false},
		{"not hexadecimal", "zzzz2f456c269b6cbb2012ad434981c8a868ee8a7ed1be8f3a75cec449d1b31f", false},
		{"too short", "6dcc2f456c269b6cbb2012ad434981c8", false},
		{"too long", "6dcc2f456c269b6cbb2012ad434981c8a868ee8a7ed1be8f3a75cec449d1b31f00", false},
		{"odd length", "6dcc2f456c269b6cbb2012ad434981c8a868ee8a7ed1be8f3a75cec449d1b31", false},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {

			result, err := DecodeMasterKey(test.masterKey)

			if !test.isValid {
				require.Error(t, err)
				return
			}

			require.Nil(t, err)
			require.Len(t, result, masterKeyLength)
		})
	}
}

func TestNewMasterKey(t *testing.T) {

	// A generated key must round-trip through DecodeMasterKey
	result, err := DecodeMasterKey(NewMasterKey())
	require.Nil(t, err)
	require.Len(t, result, masterKeyLength)

	// Two generated keys must not collide
	require.NotEqual(t, NewMasterKey(), NewMasterKey())
}

func TestNewDomain_HasValidMasterKey(t *testing.T) {

	// Every Domain built by the constructor must be able to encrypt vault data
	domain := NewDomain()

	result, err := DecodeMasterKey(domain.MasterKey)
	require.Nil(t, err)
	require.Len(t, result, masterKeyLength)
}

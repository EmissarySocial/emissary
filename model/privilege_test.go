package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestPrivilegeSchema returns the rosetta schema that describes a TestPrivilege
func TestPrivilegeSchema(t *testing.T) {

	privilege := NewPrivilege()
	s := schema.New(PrivilegeSchema())

	table := []tableTestItem{
		{"privilegeId", "123456781234567812345678", nil},
		{"identityId", "876543218765432187654321", nil},
		{"userId", "111111111111111111111111", nil},
		{"circleId", "222222222222222222222222", nil},
		{"remotePersonId", "REMOTE-PERSON", nil},
		{"remoteProductId", "REMOTE-PRODUCT", nil},
		{"remotePurchaseId", "REMOTE-PURCHASE", nil},
		{"identifierType", IdentifierTypeEmail, nil},
		{"identifierValue", "person@example.com", nil},
	}

	tableTest_Schema(t, &s, &privilege, table)
}

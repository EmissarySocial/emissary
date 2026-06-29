package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
)

func TestKeyPackageSchema(t *testing.T) {

	keyPackage := NewKeyPackage()
	s := schema.New(KeyPackageSchema())

	table := []tableTestItem{
		{"keyPackageId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"mediaType", vocab.MediaTypeMLS, nil},
		{"encoding", vocab.EncodingTypeBase64, nil},
		{"content", "BASE64-CONTENT", nil},
		{"generatorId", "GENERATOR-ID", nil},
		{"generatorName", "GENERATOR-NAME", nil},
		{"ciphersuite", "CIPHERSUITE", nil},
	}

	tableTest_Schema(t, &s, &keyPackage, table)
}

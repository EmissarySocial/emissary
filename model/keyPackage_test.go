package model

import (
	"strings"
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestKeyPackageSchema confirms that every schema property round-trips through the KeyPackage accessors
func TestKeyPackageSchema(t *testing.T) {

	keyPackage := NewKeyPackage()
	s := schema.New(KeyPackageSchema())

	table := []tableTestItem{
		{"keyPackageId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"mediaType", vocab.MediaTypeMLS, nil},
		{"encoding", vocab.EncodingTypeBase64, nil},
		{"content", "BASE64-CONTENT", nil},
		{"summary", "🕌🍉🧃🩻🥜", nil},
		{"generatorId", "GENERATOR-ID", nil},
		{"generatorName", "GENERATOR-NAME", nil},
		{"ciphersuite", "CIPHERSUITE", nil},
	}

	tableTest_Schema(t, &s, &keyPackage, table)
}

// TestKeyPackageSummaryMaxLength confirms that schema validation rejects over-limit summaries
func TestKeyPackageSummaryMaxLength(t *testing.T) {

	s := schema.New(KeyPackageSchema())

	// Build a KeyPackage that satisfies every required schema property
	keyPackage := NewKeyPackage()
	require.True(t, keyPackage.SetString("userId", "876543218765432187654321"))
	keyPackage.MediaType = vocab.MediaTypeMLS
	keyPackage.Encoding = vocab.EncodingTypeBase64
	keyPackage.Content = "BASE64-CONTENT"
	keyPackage.GeneratorID = "GENERATOR-ID"
	keyPackage.GeneratorName = "GENERATOR-NAME"
	keyPackage.Ciphersuite = "CIPHERSUITE"

	// An at-limit summary passes validation
	keyPackage.Summary = strings.Repeat("🕌", 256)
	_, err := s.Validate(&keyPackage)
	require.NoError(t, err)

	// An over-limit summary fails validation
	keyPackage.Summary = strings.Repeat("🕌", 256+1)
	_, err = s.Validate(&keyPackage)
	require.Error(t, err)
}

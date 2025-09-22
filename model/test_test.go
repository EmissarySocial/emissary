package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

type tableTestItem struct {
	property string
	input    any
	output   any
}

// testProperties is a simple table-based test using the schema package to
// of set/get all properties in an object.
func tableTest_Schema(t *testing.T, s *schema.Schema, object any, table []tableTestItem) {

	for _, test := range table {

		if test.output == nil {
			test.output = test.input
		}

		// Try to set the property
		err := s.Set(object, test.property, test.input)
		require.Nil(t, err)

		// Try to get the property
		result, err := s.Get(object, test.property)
		require.Nil(t, err)

		// Property values should be equal
		require.Equal(t, test.output, result)
	}

	// Test invalid properties
	{
		require.NotNil(t, s.Set(object, "invalid-property-that-should-never-ever-exist", "test-value"))

		result, err := s.Get(object, "invalid-property-that-should-never-ever-exist")
		require.Nil(t, result)
		require.NotNil(t, err)
	}

	// Test Validation
	require.Nil(t, s.Validate(object))
}

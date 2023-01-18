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

		require.Nil(t, s.Set(object, test.property, test.input))

		result, err := s.Get(object, test.property)
		require.Nil(t, err)
		require.Equal(t, test.output, result)
	}
}

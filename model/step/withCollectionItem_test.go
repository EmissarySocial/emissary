package step

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestWithCollectionItem verifies that a "with-collection-item" step parses its type and its sub-steps
func TestWithCollectionItem(t *testing.T) {

	step, err := NewWithCollectionItem(mapof.Any{
		"type": "Quotes",
		"steps": []any{
			mapof.Any{"do": "delete", "method": "post"},
		},
	}, nil)
	require.Nil(t, err)
	require.Equal(t, "Quotes", step.Type)
	require.Len(t, step.SubSteps, 1)
	require.Equal(t, "delete", step.SubSteps[0].Name())

	require.Equal(t, "with-collection-item", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestWithCollectionItem_TypeRequired verifies that the step refuses to load without a collection type,
// which would otherwise let it work on every kind of CollectionItem
func TestWithCollectionItem_TypeRequired(t *testing.T) {

	_, err := NewWithCollectionItem(mapof.Any{"steps": []any{}}, nil)
	require.True(t, derp.IsValidationError(err))
}

// TestWithCollectionItem_InvalidSteps verifies that a sub-step that does not parse fails the whole step
func TestWithCollectionItem_InvalidSteps(t *testing.T) {

	_, err := NewWithCollectionItem(mapof.Any{
		"type":  "Quotes",
		"steps": []any{mapof.Any{"do": "no-such-step"}},
	}, nil)
	require.NotNil(t, err)
}

// TestNew_WithCollectionItem verifies that the step is registered under its template name
func TestNewFromMap_WithCollectionItem(t *testing.T) {

	step, err := New(mapof.Any{"do": "with-collection-item", "type": "Quotes"}, nil)
	require.Nil(t, err)
	require.Equal(t, "with-collection-item", step.Name())
}

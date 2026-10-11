package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestViewJSON verifies that a "view-json" step parses its configuration
func TestViewJSON(t *testing.T) {

	step, err := NewViewJSON(mapof.Any{"value": ".Object"})
	require.Nil(t, err)
	require.NotNil(t, step.Value)
	require.Equal(t, "application/json", step.ContentType)

	// A jsonp wrapper is also accepted.
	step, err = NewViewJSON(mapof.Any{"value": ".Object", "jsonp": "callback"})
	require.Nil(t, err)
	require.NotNil(t, step.Value)

	require.Equal(t, "view-json", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestViewJSON_RequiresValue verifies that a "view-json" step requires a value
func TestViewJSON_RequiresValue(t *testing.T) {
	// A query template is required.
	_, err := NewViewJSON(mapof.Any{})
	require.NotNil(t, err)
}

// TestViewJSON_Rules covers the "rules" mode: its rules, its default schema, and a schema that extends it
func TestViewJSON_Rules(t *testing.T) {

	rules := []any{
		map[string]any{"target": "type", "value": "ArtistCredit"},
		map[string]any{"target": "id", "path": "url"},
	}

	t.Run("without a schema, every property is Any", func(t *testing.T) {

		// FUNKWHALE D17: an empty schema must not fail every rule
		step, err := NewViewJSON(mapof.Any{"rules": rules, "content-type": "application/activity+json"})
		require.NoError(t, err)
		require.Nil(t, step.Value)
		require.Len(t, step.Rules, 2)
		require.Equal(t, "application/activity+json", step.ContentType)
		require.Equal(t, schema.New(schema.Object{Properties: schema.ElementMap{}, Wildcard: schema.Any{}}), step.Schema)
	})

	t.Run("a schema extends the default", func(t *testing.T) {
		step, err := NewViewJSON(mapof.Any{"rules": rules, "schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{"index": map[string]any{"type": "integer"}},
		}})
		require.NoError(t, err)

		object, isObject := step.Schema.Element.(schema.Object)
		require.True(t, isObject)
		require.Equal(t, schema.Integer{}, object.Properties["index"])
		require.Equal(t, schema.Any{}, object.Wildcard, "undeclared properties stay allowed")
	})

	t.Run("a schema may replace the wildcard", func(t *testing.T) {
		step, err := NewViewJSON(mapof.Any{"rules": rules, "schema": mapof.Any{
			"type":     "object",
			"wildcard": map[string]any{"type": "string"},
		}})
		require.NoError(t, err)

		object, isObject := step.Schema.Element.(schema.Object)
		require.True(t, isObject)
		require.Equal(t, schema.String{}, object.Wildcard)
	})

	t.Run("an empty list of rules is allowed", func(t *testing.T) {
		step, err := NewViewJSON(mapof.Any{"rules": []any{}})
		require.NoError(t, err)
		require.Empty(t, step.Rules)
	})
}

// TestViewJSON_RulesErrors covers every configuration that must fail when the Template loads
func TestViewJSON_RulesErrors(t *testing.T) {

	rules := []any{map[string]any{"target": "type", "value": "ArtistCredit"}}

	invalid := func(name string, stepInfo mapof.Any) {
		t.Run(name, func(t *testing.T) {
			_, err := NewViewJSON(stepInfo)
			require.Error(t, err)
		})
	}

	invalid("value and rules together", mapof.Any{"value": ".Object", "rules": rules})
	invalid("a schema that is not an object", mapof.Any{"rules": rules, "schema": map[string]any{"type": "string"}})
	invalid("a schema that is not a map", mapof.Any{"rules": rules, "schema": "object"})
	invalid("an array with no maxLength", mapof.Any{"rules": rules, "schema": map[string]any{
		"type":       "object",
		"properties": map[string]any{"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
	}})
	invalid("a nested array with no maxLength", mapof.Any{"rules": rules, "schema": map[string]any{
		"type": "object",
		"properties": map[string]any{"credits": map[string]any{
			"type": "array", "maxLength": 4,
			"items": map[string]any{"type": "object", "properties": map[string]any{
				"names": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}},
		}},
	}})
	invalid("a wildcard array with no maxLength", mapof.Any{"rules": rules, "schema": map[string]any{
		"type":     "object",
		"wildcard": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}})
	invalid("a rule that does not parse", mapof.Any{"rules": []any{map[string]any{"target": "x"}}})
}

// TestArrayWithoutMaxLength names the first unbounded array, and passes a schema with none
func TestArrayWithoutMaxLength(t *testing.T) {

	bounded := schema.Object{Properties: schema.ElementMap{
		"tags":  schema.Array{Items: schema.String{}, MaxLength: 8},
		"name":  schema.String{},
		"inner": schema.Object{Properties: schema.ElementMap{"list": schema.Array{Items: schema.Any{}, MaxLength: 1}}},
	}}

	_, found := arrayWithoutMaxLength(bounded, "")
	require.False(t, found)

	unbounded := schema.Object{Properties: schema.ElementMap{
		"inner": schema.Object{Properties: schema.ElementMap{"list": schema.Array{Items: schema.Any{}}}},
	}}

	path, found := arrayWithoutMaxLength(unbounded, "")
	require.True(t, found)
	require.Equal(t, "inner.list", path)

	path, found = arrayWithoutMaxLength(schema.Array{Items: schema.Array{Items: schema.String{}}, MaxLength: 2}, "")
	require.True(t, found)
	require.Equal(t, ".*", path)
}

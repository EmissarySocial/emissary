package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// parseStep unmarshals a step definition exactly the way the Template service does, so that
// these tests exercise the same value types production sees. This matters: schema.UnmarshalMap
// type-asserts each nested value to map[string]any, and a hand-built mapof.Any fails that
// assertion even though it has the same underlying type.
func parseStep(t *testing.T, definition string) mapof.Any {

	t.Helper()

	result := mapof.NewAny()
	require.NoError(t, hjson.Unmarshal([]byte(definition), &result))

	return result
}

// TestReadForm verifies that a "read-form" step parses its schema
func TestReadForm(t *testing.T) {

	step, err := NewReadForm(parseStep(t, `{
		do: read-form
		schema: {
			type: object
			properties: {
				name: {type:"string", maxLength:128, required:true}
				email: {type:"string", format:"email", maxLength:255, required:true}
				message: {type:"string", maxLength:4096, required:true}
			}
		}
	}`))

	require.NoError(t, err)
	require.Equal(t, "read-form", step.Name())

	for _, field := range []string{"name", "email", "message"} {
		element, exists := step.Schema.GetElement(field)
		require.True(t, exists, "schema is missing %q", field)
		require.NotNil(t, element)
	}
}

// TestReadForm_BoundsAreLoaded verifies that the schema's length bound survives parsing. The
// bound is the whole point of the step (D10) -- an unbounded field is how an oversized message
// reaches a header, so a schema that parsed but dropped its limits would be worse than none.
func TestReadForm_BoundsAreLoaded(t *testing.T) {

	step, err := NewReadForm(parseStep(t, `{
		schema: {type:"object", properties:{name:{type:"string", maxLength:10}}}
	}`))

	require.NoError(t, err)

	values := mapof.NewAny()
	require.NoError(t, step.Schema.Set(&values, "name", "12345678901234567890"))
	require.Equal(t, 10, len(values.GetString("name")), "maxLength was not enforced")
}

// TestReadForm_RequiresSchema verifies that a step with no schema is rejected at load time.
// The schema is the only thing bounding visitor input, so an unbounded step is never valid.
func TestReadForm_RequiresSchema(t *testing.T) {

	_, err := NewReadForm(mapof.Any{})

	require.ErrorContains(t, err, "requires a 'schema'")
}

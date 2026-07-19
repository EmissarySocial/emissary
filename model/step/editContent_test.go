package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestEditContent(t *testing.T) {

	step, err := NewEditContent(mapof.Any{
		"file":   "my-file",
		"field":  "body",
		"format": "HTML",
	})
	require.Nil(t, err)
	require.Equal(t, "my-file", step.Filename)
	require.Equal(t, "body", step.Fieldname)
	require.Equal(t, "HTML", step.Format)

	require.Equal(t, "edit-content", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

func TestEditContent_Defaults(t *testing.T) {

	// "field" defaults to "content", "format" defaults to "editorjs".
	step, err := NewEditContent(mapof.Any{"format": "HTML"})
	require.Nil(t, err)
	require.Equal(t, "content", step.Fieldname)

	// "max-length" defaults to the per-step default (in KB, converted to runes) when unset.
	require.Equal(t, editContentDefaultMaxLengthKB*runesPerKilobyte, step.MaxLength)
}

func TestEditContent_MaxLength(t *testing.T) {

	// An explicit, in-range "max-length" (in KB) is converted to runes.
	step, err := NewEditContent(mapof.Any{"format": "HTML", "max-length": 100})
	require.Nil(t, err)
	require.Equal(t, 100*runesPerKilobyte, step.MaxLength)
}

func TestEditContent_MaxLength_DefaultsWhenZeroOrNegative(t *testing.T) {

	// A zero or negative "max-length" falls back to the default.
	step, err := NewEditContent(mapof.Any{"format": "HTML", "max-length": 0})
	require.Nil(t, err)
	require.Equal(t, editContentDefaultMaxLengthKB*runesPerKilobyte, step.MaxLength)

	step, err = NewEditContent(mapof.Any{"format": "HTML", "max-length": -5})
	require.Nil(t, err)
	require.Equal(t, editContentDefaultMaxLengthKB*runesPerKilobyte, step.MaxLength)
}

func TestEditContent_MaxLength_ClampedToCeiling(t *testing.T) {

	// A "max-length" (KB) larger than the storage ceiling is clamped down, so a template
	// can never allow more content than the schema will persist.
	step, err := NewEditContent(mapof.Any{"format": "HTML", "max-length": editContentMaxLengthCeilingKB + 1})
	require.Nil(t, err)
	require.Equal(t, editContentMaxLengthCeilingKB*runesPerKilobyte, step.MaxLength)
}

func TestEditContent_InvalidFormat(t *testing.T) {

	// "format" is a required enum; an invalid value fails schema validation.
	_, err := NewEditContent(mapof.Any{"format": "not-a-format"})
	require.NotNil(t, err)
}

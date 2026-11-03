package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRequire verifies that a "require" step parses its configuration
func TestRequire(t *testing.T) {

	step, err := NewRequire(mapof.Any{"condition": "{{.IsAuthor}}"}, nil)
	require.NoError(t, err)
	require.NotNil(t, step.Condition)

	require.Equal(t, "require", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestRequire_MissingCondition verifies that a step with nothing to check is refused at load time,
// rather than refusing every request
func TestRequire_MissingCondition(t *testing.T) {

	refuses := func(stepInfo mapof.Any) {
		t.Helper()
		_, err := NewRequire(stepInfo, nil)
		require.Error(t, err, "stepInfo %v", stepInfo)
	}

	refuses(mapof.Any{})
	refuses(mapof.Any{"condition": ""})
	refuses(mapof.Any{"condition": " \n\t "})
	refuses(mapof.Any{"conditon": "{{.IsAuthor}}"})
}

// TestRequire_InvalidCondition verifies that a condition that is not a valid template is refused
func TestRequire_InvalidCondition(t *testing.T) {
	_, err := NewRequire(mapof.Any{"condition": "{{ .Unclosed"}, nil)
	require.Error(t, err)
}

// TestRequire_UsesFuncMap verifies that a condition may call the helper functions it is given
func TestRequire_UsesFuncMap(t *testing.T) {

	condition := "{{in .Layout `notes` `podcast`}}"

	_, err := NewRequire(mapof.Any{"condition": condition}, nil)
	require.Error(t, err, "`in` is undefined without the funcMap")

	_, err = NewRequire(mapof.Any{"condition": condition}, testFuncMap())
	require.NoError(t, err)
}

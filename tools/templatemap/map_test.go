package templatemap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplateMap(t *testing.T) {

	// Create a new Map
	m := Map{}
	data := map[string]string{
		"value": "world",
	}

	// Unmarshal some JSON
	err := m.UnmarshalJSON([]byte(`{"hello":"{{.value}}"}`))
	require.Nil(t, err)

	// Execute a template
	success := m.Execute("hello", data)
	require.Equal(t, "world", success)

	missing := m.Execute("missing-template", data)
	require.Equal(t, "", missing)
}

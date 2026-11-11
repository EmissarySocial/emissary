package build

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// hxValsWithAction matches an `hx-vals` attribute that holds a template action
var hxValsWithAction = regexp.MustCompile(`hx-vals=('[^']*\{\{[^']*'|"[^"]*\{\{[^"]*")`)

// TestTemplates_NoFederatedValueInHxVals confirms the folders that draw federated content write no
// template value into `hx-vals`, where HTML escaping does not protect the JSON (see _embed/templates/AGENTS.md)
func TestTemplates_NoFederatedValueInHxVals(t *testing.T) {

	for _, folder := range []string{"base-social", "user-inbox", "user-notifications", "stream-outbox-message"} {

		paths, err := filepath.Glob(filepath.Join("..", "_embed", "templates", folder, "*.html"))
		require.NoError(t, err)
		require.NotEmpty(t, paths, folder)

		for _, path := range paths {
			content, err := os.ReadFile(path) // #nosec G304 -- a path globbed from this repository's own templates
			require.NoError(t, err)
			require.Empty(t, hxValsWithAction.FindAllString(string(content), -1), path)
		}
	}
}

// TestHxValsWithAction pins what the pattern finds
func TestHxValsWithAction(t *testing.T) {
	require.True(t, hxValsWithAction.MatchString(`hx-vals='{"type":"Like", "url":"{{$url}}"}'`))
	require.True(t, hxValsWithAction.MatchString(`hx-vals="{'stateId':{{$stateID}}}"`))
	require.False(t, hxValsWithAction.MatchString(`hx-vals='{"type":"Like", "exists":true}'`))
	require.False(t, hxValsWithAction.MatchString(`hx-post="/x?url={{$url | urlquery}}" hx-vals='{"a":1}'`))
}

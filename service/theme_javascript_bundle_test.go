package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestThemeJavascript_FilesEndWithSemicolon pins the rule that keeps a javascript bundle working.
// A bundle joins its files with nothing between them, so a file ending in `})()` followed by one
// starting with `(function` becomes a call on the first file's result, which throws and stops
// every script after it.
func TestThemeJavascript_FilesEndWithSemicolon(t *testing.T) {

	files, err := filepath.Glob("../_embed/templates/*/javascript/*.js")
	require.NoError(t, err)
	require.NotEmpty(t, files, "the glob must find the bundled javascript files")

	for _, filename := range files {
		content, err := os.ReadFile(filename) // #nosec G304 -- the name comes from a fixed glob over embedded templates
		require.NoError(t, err)

		trimmed := strings.TrimSpace(string(content))
		require.True(t, strings.HasSuffix(trimmed, ";"), "%s must end with a semicolon", filename)
	}
}

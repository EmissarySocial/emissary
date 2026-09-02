package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// validatorOption matches the "validator" option wherever a shipped hjson definition declares one,
// capturing the URL it points at.  hjson quoting is uniform across these files, so a regex over the
// source is enough and avoids standing up the Template and Widget services just to read a string.
var validatorOption = regexp.MustCompile(`validator:\s*"([^"]+)"`)

// TestValidatorURLsAreRouted asserts that every server-side field validator declared by a shipped
// Template or Widget names a GET route that actually exists.
//
// These URLs are strings in hjson that nothing else checks.  The client end is a hyperscript
// behavior that fetches the URL and reads result.valid off the response; point it at a path with no
// route and it collects whatever that path does return, fails to parse it as JSON, and simply never
// draws a badge.  So a renamed or deleted validate route does not break the form loudly -- it
// silently stops validating it, and the field goes on accepting anything.
//
// Matching the route PATTERN is the whole test.  A mistyped "/.validate/..." path does not even
// 404: it has two segments, so it falls through to the "/:stream/:action" catch-all and is answered
// by the Stream builder.  Asserting merely that SOME route matched would pass for every possible
// typo.  Every validator path is a literal with no parameters, so the pattern must equal the path.
func TestValidatorURLsAreRouted(t *testing.T) {

	routes := makeTestRoutes()
	root := "_embed/templates"

	found := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if entry.IsDir() || filepath.Ext(path) != ".hjson" {
			return nil
		}

		definition, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		for _, match := range validatorOption.FindAllStringSubmatch(string(definition), -1) {

			// A Template's validator URL may carry template expressions in its query string
			// ("?streamId={{.StreamID}}"), which StepEditModelObject renders at build time.
			// Only the path ahead of the "?" is routed, and only it can be checked here.
			url, _, _ := strings.Cut(match[1], "?")
			found++

			require.Equal(t, url, matchedGetRoute(routes, url),
				"%s declares validator %q, which is not registered as a GET route of its own", path, url)
		}

		return nil
	})

	require.NoError(t, err)
	require.NotZero(t, found, "the scan found no validators at all, so it is asserting nothing")
}

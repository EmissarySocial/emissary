package tests

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestTagPaths_AreStrings verifies that every tagPaths entry of every template is a string
// element of the schema its builder edits with, in every group that is checked out.
func TestTagPaths_AreStrings(t *testing.T) {

	// A tag path is linked to its schema by nothing but this test; see "A tagPaths entry and
	// the schema element behind it are linked by nothing but a test" in _embed/templates/AGENTS.md.

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			checked := 0

			for _, templateID := range loaded.templateIDs() {

				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				for _, path := range template.TagPaths {
					for _, context := range templateContexts(loaded, template) {

						where := templateID

						if context.Label != "" {
							where += " " + context.Label
						}

						element, found := context.Schema.GetElement(path)

						// RULE: The save reads the path through the schema, so an entry it does not resolve is dead
						require.True(t, found, "%s: tagPaths entry %q names no element of the schema, so the save finds nothing there and a form cannot write to it; declare it as a string or drop the entry", where, path)

						// RULE: A path the schema does not declare falls through to a wildcard, and a form
						// value posted there is stored as a one-element array that reads back empty
						require.IsType(t, schema.String{}, element, "%s: tagPaths entry %q is %T in the schema, so a form value posted to it is stored as a one-element array that reads back empty after a database round trip; declare it as a string", where, path, element)
						checked++
					}
				}
			}

			require.Positive(t, checked, "no template in %s declares tagPaths -- this test is watching nothing", group.Name)
		})
	}
}

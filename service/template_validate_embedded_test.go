package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedTemplates_Validate runs the REAL validateTemplates() pass over every template that
// ships in the binary, in the same order the loader does: add all, calculate inheritance, then
// validate.  The other whole-tree tests only unmarshal a definition, so nothing exercised the
// cross-template rules -- undefined states and roles, a form field with no schema property, or a
// send-email step that names a missing email or omits a key that email requires.  Each of those
// currently surfaces the first time a server boots with the template, or later still.
//
// Note that form fields naming a BASE-schema property (token, label, summary) resolve whether or
// not the template redeclares them, because Template.Add inherits the model's schema.  What this
// catches is a field naming a property that exists nowhere -- a typo, or a "data.*" value the
// template forgot to declare.
func TestEmbeddedTemplates_Validate(t *testing.T) {

	const root = "../_embed/templates"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	emailService := testServerEmail()

	templateService := &Template{
		templatePrep: make(set.Map[model.Template]),
		funcMap:      emissarytemplates.FuncMap(nullIconProvider{}),
		emailService: &emailService,
	}

	templates := 0
	emails := 0

	// Load every shipped definition into the prep area, exactly as Refresh does
	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}

		filesystem := os.DirFS(filepath.Join(root, entry.Name()))
		definitionType, definition := findDefinition(filesystem)

		switch definitionType {

		case DefinitionTemplate:
			require.NoError(t, templateService.Add(entry.Name(), filesystem, definition), "template %q does not load", entry.Name())
			templates++

		case DefinitionEmail:
			require.NoError(t, emailService.Add(filesystem, definition), "email %q does not load", entry.Name())
			emails++
		}
	}

	// A silent zero would make this test pass forever if the directory ever moved
	require.Positive(t, templates, "no templates found under "+root)
	require.Positive(t, emails, "no email definitions found under "+root)

	require.NoError(t, templateService.calculateAllInheritance())

	// Report every failure at once, the way the loader does -- one template's mistake should not
	// hide the next one's
	errors := templateService.validateTemplates()

	for _, err := range errors {
		t.Errorf("template validation failed: %s", err.Error())
	}

	require.Empty(t, errors, "%d shipped templates failed validation", len(errors))
}

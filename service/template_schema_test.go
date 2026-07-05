package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/schema"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// forEachEmbeddedTemplate parses every template definition under _embed/templates
// (template.hjson / widget.hjson / registration.hjson) the same way the template
// loader does, and invokes fn once per template inside its own subtest. It fails
// if no definitions were found, so a broken walk cannot pass silently.
func forEachEmbeddedTemplate(t *testing.T, fn func(t *testing.T, templateID string, tmpl model.Template)) {

	root := "../_embed/templates"

	entries, err := os.ReadDir(root)
	require.NoError(t, err, "unable to read embedded templates directory")

	checked := 0

	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}

		// A template definition lives in template.hjson / widget.hjson / registration.hjson
		var definition []byte
		var templateID = entry.Name()

		for _, name := range []string{"template.hjson", "widget.hjson", "registration.hjson"} {
			path := filepath.Join(root, templateID, name)
			if data, err := os.ReadFile(path); err == nil {
				definition = data
				break
			}
		}

		if definition == nil {
			continue // directory without a recognized definition file
		}

		checked++

		t.Run(templateID, func(t *testing.T) {
			tmpl := model.NewTemplate(templateID, nil)
			require.NoError(t, hjson.Unmarshal(definition, &tmpl), "unable to unmarshal %s", templateID)
			fn(t, templateID, tmpl)
		})
	}

	require.Positive(t, checked, "expected to check at least one embedded template")
}

// TestEmbeddedTemplates_NoOrphanSchemaProperties walks every shipping template under
// _embed/templates and asserts that none of them declare a schema property that its
// model object cannot back.  This is the load-time guard that would have caught the
// bandwagon-news "title" and "content.type" bugs -- run against the real templates so a
// regression fails the build instead of a user's save.
func TestEmbeddedTemplates_NoOrphanSchemaProperties(t *testing.T) {

	forEachEmbeddedTemplate(t, func(t *testing.T, templateID string, tmpl model.Template) {

		// Reproduce the relevant part of service.Template.Add: inherit the model's
		// BaseSchema the same way the loader does.
		if tmpl.TemplateRole != "registration" {
			tmpl.Schema.Inherit(schema.New(tmpl.BaseSchema()))
		}

		orphans := tmpl.UnsupportedSchemaProperties()
		require.Empty(t, orphans,
			"template %q (model %q) declares schema properties with no model accessor: %s",
			templateID, tmpl.Model, strings.Join(orphans, ", "))
	})
}

// TestEmbeddedTemplates_ValidSchemaFormats asserts that every format name declared in
// a shipping template's schema resolves in the rosetta format registry. String
// validation silently skips unrecognized format names (degrading to the no-html
// default), so without this gate a typo'd format ships with no validation at all.
func TestEmbeddedTemplates_ValidSchemaFormats(t *testing.T) {

	forEachEmbeddedTemplate(t, func(t *testing.T, templateID string, tmpl model.Template) {
		require.NoError(t, tmpl.Schema.ValidateFormats(),
			"template %q declares an unrecognized schema format name", templateID)
	})
}

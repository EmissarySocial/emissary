package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/set"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedTemplates_MenubarActions confirms that Promote is the single control that takes an
// article live, on every variant, and that its `promote-draft` really does publish.
//
// RULE: promoting is the ONLY path to `save-and-publish`, and `save-and-publish` is the only thing
// that moves a Stream inside withinPublishDate() -- which Common.Navigation ANDs in with NO
// domain-owner bypass.  A variant that overrides `promote-draft` away therefore cannot reach
// auto-generated navigation at all, for anyone, and nothing else reports that.
func TestEmbeddedTemplates_MenubarActions(t *testing.T) {

	const root = "../_embed/templates"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	emailService := testServerEmail()

	templateService := &Template{
		templatePrep: make(set.Map[model.Template]),
		funcMap:      emissarytemplates.FuncMap(nullIconProvider{}),
		emailService: &emailService,
	}

	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}

		filesystem := os.DirFS(filepath.Join(root, entry.Name()))
		definitionType, definition := findDefinition(filesystem)

		if definitionType == DefinitionTemplate {
			require.NoError(t, templateService.Add(entry.Name(), filesystem, definition), "template %q does not load", entry.Name())
		}
	}

	require.NoError(t, templateService.calculateAllInheritance())

	// menubar returns the source of the `edit-menubar` that the named Template actually renders,
	// after inheritance has chosen between its own copy and its parent's
	menubar := func(templateID string) string {

		template, exists := templateService.templatePrep[templateID]
		require.True(t, exists, "template %q is not loaded", templateID)

		partial := template.HTMLTemplate.Lookup("edit-menubar")
		require.NotNil(t, partial, "template %q has no edit-menubar", templateID)

		return partial.Tree.Root.String()
	}

	for _, templateID := range []string{"article-base", "article-editorjs", "article-html", "article-markdown", "article-remote", "article-two-column"} {

		source := menubar(templateID)

		// The chrome must still be able to report an inline failure, and Delete must survive
		require.Contains(t, source, "htmx-response-message", "%s must keep the inline error target", templateID)
		require.Contains(t, source, "delete", "%s must still offer Delete", templateID)

		// One button takes an article live, and it is this one
		require.Contains(t, source, "saveAndPromote()", "%s must offer Promote", templateID)
		require.Contains(t, source, "discard-draft", "%s must offer Discard Draft", templateID)

		// The draft workflow is what Promote drives, so the action behind it has to publish
		require.True(t, promotePublishes(t, templateService, templateID),
			"%s: promote-draft must run save-and-publish, or Promote cannot take the article live", templateID)
	}
}

// promotePublishes reports whether the named Template's `promote-draft` action, as inheritance
// resolved it, still contains the save-and-publish step that takes a Stream live.
func promotePublishes(t *testing.T, templateService *Template, templateID string) bool {

	t.Helper()

	template, exists := templateService.templatePrep[templateID]
	require.True(t, exists, "template %q is not loaded", templateID)

	action, exists := template.Actions["promote-draft"]
	require.True(t, exists, "template %q has no promote-draft action", templateID)

	for _, item := range action.Steps {
		if _, ok := item.(step.SaveAndPublish); ok {
			return true
		}
	}

	return false
}

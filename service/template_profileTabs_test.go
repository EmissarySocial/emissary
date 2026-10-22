package service

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/set"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// profileTabsBuilder is the smallest dot that the base-outbox Profile tabs render against
type profileTabsBuilder struct {
	action string
}

// IsAction returns TRUE if this stub is rendering the named action
func (builder profileTabsBuilder) IsAction(actionID string) bool {
	return builder.action == actionID
}

// IsDesktop returns TRUE, so that the tabs render their desktop layout
func (builder profileTabsBuilder) IsDesktop() bool {
	return true
}

// View returns a marker naming the embedded action, in place of rendering its form
func (builder profileTabsBuilder) View(actionID string) (template.HTML, error) {
	return template.HTML("[view:" + actionID + "]"), nil // #nosec G203 -- a fixed test marker
}

// loadProfileTabsTemplates loads every embedded Template plus any extra folders, and calculates inheritance
func loadProfileTabsTemplates(t *testing.T, extra map[string]fstest.MapFS) *Template {

	t.Helper()

	const root = "../_embed/templates"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	emailService := testServerEmail()

	templateService := &Template{
		templatePrep: make(set.Map[model.Template]),
		funcMap:      emissarytemplates.FuncMap(nullIconProvider{}),
		emailService: &emailService,
	}

	// Load the embedded Templates
	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}

		filesystem := os.DirFS(filepath.Join(root, entry.Name()))

		if definitionType, definition := findDefinition(filesystem); definitionType == DefinitionTemplate {
			require.NoError(t, templateService.Add(entry.Name(), filesystem, definition), "template %q does not load", entry.Name())
		}
	}

	// Load the extra Templates that this test defines
	for name, filesystem := range extra {
		definitionType, definition := findDefinition(filesystem)
		require.Equal(t, DefinitionTemplate, definitionType, "extra template %q has no definition", name)
		require.NoError(t, templateService.Add(name, filesystem, definition), "extra template %q does not load", name)
	}

	require.NoError(t, templateService.calculateAllInheritance())
	return templateService
}

// renderProfileTab executes one Profile tab of the named Template against the stub builder
func renderProfileTab(t *testing.T, templateService *Template, templateID string, actionID string) string {

	t.Helper()

	template, exists := templateService.templatePrep[templateID]
	require.True(t, exists, "template %q is not loaded", templateID)

	var buffer bytes.Buffer
	require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, actionID, profileTabsBuilder{action: actionID}))

	return buffer.String()
}

// TestProfileTabs_UserOutboxInheritsBaseOutbox confirms that user-outbox serves every Profile tab from base-outbox
func TestProfileTabs_UserOutboxInheritsBaseOutbox(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	userOutbox := templateService.templatePrep["user-outbox"]

	for _, actionID := range []string{
		"settings-general", "settings-general-form",
		"settings-images", "settings-images-form",
		"settings-links", "settings-links-form",
		"settings-icon-delete", "settings-image-delete",
	} {
		_, exists := userOutbox.Actions[actionID]
		require.True(t, exists, "user-outbox is missing %q", actionID)
	}

	require.NotNil(t, userOutbox.HTMLTemplate.Lookup("settings-tabs"))

	// Each tab marks only itself as selected, and embeds its own form
	for _, tab := range []string{"general", "images", "links"} {

		html := renderProfileTab(t, templateService, "user-outbox", "settings-"+tab)

		require.Contains(t, html, `hx-get="/@me/settings-`+tab+`" hx-target="#profile-settings" hx-push-url="false" class="turboclick selected"`)
		require.Contains(t, html, "[view:settings-"+tab+"-form]")
		require.NotContains(t, html, "hx-boost=\"true\">General", "a tab must not push the URL")
	}
}

// TestProfileTabs_SearchIndex confirms that the profile forms that change search results re-index the user
func TestProfileTabs_SearchIndex(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	baseOutbox := templateService.templatePrep["base-outbox"]

	hasSearchIndex := func(actionID string) bool {
		for _, item := range baseOutbox.Actions[actionID].Steps {
			if _, ok := item.(step.SearchIndex); ok {
				return true
			}
		}
		return false
	}

	for _, actionID := range []string{"settings-general-form", "settings-images-form", "settings-icon-delete", "settings-image-delete"} {
		require.True(t, hasSearchIndex(actionID), "%s must run search-index", actionID)
	}

	require.False(t, hasSearchIndex("settings-links-form"), "links do not appear in search results")
}

// TestProfileTabs_OverrideTabBar confirms that an outbox's own settings-tabs replaces the bar on every inherited tab
func TestProfileTabs_OverrideTabBar(t *testing.T) {

	custom := fstest.MapFS{
		"template.hjson": &fstest.MapFile{Data: []byte(`{
			templateId: test-outbox
			templateRole: user-outbox
			category: Special
			model: Outbox
			extends: ["base-outbox"]
			states: {
				default: {
					label: "Default"
				}
			}
			actions: {}
		}`)},
		"settings-tabs.html": &fstest.MapFile{Data: []byte(`<nav id="custom-tabs">{{if .IsAction "settings-images"}}IMAGES-SELECTED{{end}}</nav>`)},
	}

	templateService := loadProfileTabsTemplates(t, map[string]fstest.MapFS{"test-outbox": custom})

	// The custom bar replaces the stock one on all three inherited tabs
	for _, tab := range []string{"general", "images", "links"} {

		html := renderProfileTab(t, templateService, "test-outbox", "settings-"+tab)

		require.Contains(t, html, `<nav id="custom-tabs">`)
		require.NotContains(t, html, `id="menu-bar"`, "the stock tab bar must not render")
		require.Contains(t, html, "[view:settings-"+tab+"-form]", "the inherited form must still render")
	}

	require.Contains(t, renderProfileTab(t, templateService, "test-outbox", "settings-images"), "IMAGES-SELECTED")
}

// TestProfileTabs_SettingsShell confirms that the Profile settings page is a shell, and the old tabs redirect to it
func TestProfileTabs_SettingsShell(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	userSettings := templateService.templatePrep["user-settings"]

	// The shell loads the outbox's General tab, and carries no tabs or forms of its own
	general, err := os.ReadFile("../_embed/templates/user-settings/general.html")
	require.NoError(t, err)

	require.Contains(t, string(general), `hx-get="/@me/settings-general"`)
	require.NotContains(t, string(general), "menu-bar")
	require.NotContains(t, string(general), ".View")

	// The old forms are gone, and the old tabs redirect to the shell
	for _, actionID := range []string{"general-form", "images-form", "links-form"} {
		_, exists := userSettings.Actions[actionID]
		require.False(t, exists, "user-settings must not define %q", actionID)
	}

	for _, actionID := range []string{"images", "links"} {

		action := userSettings.Actions[actionID]
		require.Len(t, action.Steps, 1)

		redirect, ok := action.Steps[0].(step.RedirectTo)
		require.True(t, ok, "%s must be a redirect-to", actionID)
		require.Equal(t, "/@me/settings/general", redirect.URL.Root.String())
	}
}

// TestProfileTabs_MissingTabsReported confirms that an outbox without base-outbox is reported, and nothing else is
func TestProfileTabs_MissingTabsReported(t *testing.T) {

	bare := fstest.MapFS{
		"template.hjson": &fstest.MapFile{Data: []byte(`{
			templateId: bare-outbox
			templateRole: user-outbox
			category: Special
			model: Outbox
			states: {
				default: {
					label: "Default"
				}
			}
			actions: {}
		}`)},
	}

	templateService := loadProfileTabsTemplates(t, map[string]fstest.MapFS{"bare-outbox": bare})
	errs := templateService.missingProfileTabs()

	require.Len(t, errs, 1, "only bare-outbox lacks the Profile tabs")
	require.Contains(t, derp.Message(errs[0]), "base-outbox")
	require.Contains(t, fmt.Sprint(derp.Details(errs[0])), "bare-outbox")
	require.Contains(t, fmt.Sprint(derp.Details(errs[0])), "settings-general")
}

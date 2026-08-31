package service

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/set"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// dataStub stands in for the Stream builder, supplying just the accessor that a layout
// control reads
type dataStub map[string]string

// DataString returns a stubbed data value, mirroring build.Stream.DataString
func (stub dataStub) DataString(key string) string {
	return stub[key]
}

// loadEmbeddedTemplates parses every shipped template through the real loader and resolves
// inheritance, so a test sees the same parse trees a running server does.
func loadEmbeddedTemplates(t *testing.T) *Template {
	t.Helper()

	root := "../_embed/templates"
	emailService := testServerEmail()

	templateService := &Template{
		templatePrep: make(set.Map[model.Template]),
		funcMap:      emissarytemplates.FuncMap(nullIconProvider{}),
		emailService: &emailService,
	}

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

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

	return templateService
}

// TestLayoutControls_Slot renders the layout-controls slot of every shipped Template that
// extends base-widget-editor, against the REAL parse trees after inheritance.
//
// This is the whole opt-in chain in one assertion.  The base ships an empty default plus a
// reusable width control; a Template opts in by defining a layout-controls.html of its own,
// which may call the shared control (contact form, article) or supply a different one entirely
// (folder, whose page shape is how its children are listed, not a width).  A Template that has
// not opted in must render nothing at all.
//
// The width binding is checked separately from the control: only .widget-editor-width may drive
// the canvas proportions, so a Template adding some other setting here cannot resize it.
func TestLayoutControls_Slot(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	tests := []struct {
		templateID    string
		field         string // "" means this Template has not opted in
		stored        map[string]string
		selected      string
		resizesCanvas bool
	}{
		// Keyed by templateId from the hjson, NOT by directory name
		{templateID: "contact-form", field: "data.width", stored: map[string]string{"width": "MEDIUM"}, selected: "MEDIUM", resizesCanvas: true},
		{templateID: "article-base", field: "data.width", stored: map[string]string{"width": "SMALL"}, selected: "SMALL", resizesCanvas: true},
		{templateID: "folder", field: "data.format", stored: map[string]string{"format": "CARDS"}, selected: "CARDS", resizesCanvas: false},
		{templateID: "base-widget-editor", field: ""},
	}

	for _, test := range tests {

		template, exists := templateService.templatePrep[test.templateID]
		require.True(t, exists, "template %q not found", test.templateID)

		var buffer strings.Builder
		require.NoError(t,
			template.HTMLTemplate.ExecuteTemplate(&buffer, "layout-controls", dataStub(test.stored)),
			"template %q has no layout-controls slot", test.templateID)

		output := buffer.String()

		if test.field == "" {
			require.Empty(t, strings.TrimSpace(output), "template %q must render an empty slot", test.templateID)
			continue
		}

		require.Contains(t, output, `name="`+test.field+`"`, "template %q", test.templateID)
		require.Contains(t, output, `value="`+test.selected+`" selected>`, "template %q", test.templateID)
		require.Equal(t, 1, strings.Count(output, " selected>"), "template %q: exactly one option must be selected", test.templateID)

		require.Equal(t, test.resizesCanvas, strings.Contains(output, "widget-editor-width"),
			"template %q: only the width control may carry the class that re-proportions the canvas", test.templateID)
	}
}

// TestLayoutControls_Selection asserts that exactly one option is selected for every value a
// Template might have stored, including none.
//
// An unset value is folded into the option that already matches what the page renders: an empty
// width produces class "layout-", which matches no rule and fills the width exactly like LARGE,
// and an empty format falls through view.html's final "else" to the table layout.  Saying so in
// the markup is what keeps the control honest AND keeps the field present in every POST.
func TestLayoutControls_Selection(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	tests := []struct {
		templateID string
		key        string
		stored     string
		selected   string
	}{
		{templateID: "contact-form", key: "width", stored: "LARGE", selected: "LARGE"},
		{templateID: "contact-form", key: "width", stored: "MEDIUM", selected: "MEDIUM"},
		{templateID: "contact-form", key: "width", stored: "SMALL", selected: "SMALL"},
		{templateID: "contact-form", key: "width", stored: "", selected: "LARGE"},
		{templateID: "contact-form", key: "width", stored: "garbage", selected: "LARGE"},

		{templateID: "folder", key: "format", stored: "TABLE", selected: "TABLE"},
		{templateID: "folder", key: "format", stored: "CARDS", selected: "CARDS"},
		{templateID: "folder", key: "format", stored: "COLUMNS", selected: "COLUMNS"},
		{templateID: "folder", key: "format", stored: "", selected: "TABLE"},
		{templateID: "folder", key: "format", stored: "garbage", selected: "TABLE"},
	}

	for _, test := range tests {

		template, exists := templateService.templatePrep[test.templateID]
		require.True(t, exists, "template %q not found", test.templateID)

		var buffer strings.Builder
		require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, "layout-controls", dataStub{test.key: test.stored}))
		output := buffer.String()

		require.Equal(t, 3, strings.Count(output, "<option "), "%s stored %q", test.templateID, test.stored)
		require.Equal(t, 1, strings.Count(output, " selected>"),
			"%s stored %q: exactly one option must be selected", test.templateID, test.stored)
		require.Contains(t, output, `value="`+test.selected+`" selected>`, "%s stored %q", test.templateID, test.stored)
	}
}

// TestLayoutControls_WidgetsActionSaves asserts that every Template shipping a layout control
// also reads that control's field back out of the widget editor's form.  The control and the
// save are wired through two different files, so a rename in either one would otherwise leave
// the control silently inert -- it would look like it worked and change nothing.
func TestLayoutControls_WidgetsActionSaves(t *testing.T) {

	tests := []struct {
		directory string
		field     string
	}{
		{directory: "stream-contact-form", field: "data.width"},
		{directory: "stream-article-base", field: "data.width"},
		{directory: "stream-folder", field: "data.format"},
	}

	for _, test := range tests {

		definition, err := os.ReadFile("../_embed/templates/" + test.directory + "/template.hjson")
		require.NoError(t, err)

		template := model.NewTemplate(test.directory, nil)
		require.NoError(t, hjson.Unmarshal(definition, &template))

		action, exists := template.Action("widgets")
		require.True(t, exists, "%s must define a widgets action", test.directory)

		require.True(t, savesField(action.Steps, test.field),
			`%s: the widgets action must include {do:"set-data", from-form:["%s"]}`, test.directory, test.field)
	}
}

// savesField returns TRUE if a pipeline reads the named field from the posted form, following
// any nested pipeline (article templates wrap their widget editor in "with-draft").
func savesField(steps []step.Step, field string) bool {

	for _, item := range steps {

		if setData, ok := item.(step.SetData); ok {
			if slices.Contains(setData.FromForm, field) {
				return true
			}
		}

		if withDraft, ok := item.(step.WithDraft); ok {
			if savesField(withDraft.SubSteps, field) {
				return true
			}
		}
	}

	return false
}

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
	"github.com/benpate/rosetta/schema"
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
	require.NoError(t, templateService.calculateAccessLists())

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
		// The base's own slot is no longer empty: it ships the width control as the default,
		// so a Template opts OUT by overriding the slot rather than opting in by defining it.
		{templateID: "base-widget-editor", field: "data.width", stored: map[string]string{"width": "LARGE"}, selected: "LARGE", resizesCanvas: true},
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
// width produces class "layout-", which matches no rule and fills the width exactly like FULL,
// and an empty format falls through view.html's final "else" to the table layout.  Saying so in
// the markup is what keeps the control honest AND keeps the field present in every POST.
func TestLayoutControls_Selection(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	tests := []struct {
		templateID string
		key        string
		stored     string
		selected   string
		options    int
	}{
		{templateID: "contact-form", key: "width", stored: "FULL", selected: "FULL", options: 4},
		{templateID: "contact-form", key: "width", stored: "LARGE", selected: "LARGE", options: 4},
		{templateID: "contact-form", key: "width", stored: "MEDIUM", selected: "MEDIUM", options: 4},
		{templateID: "contact-form", key: "width", stored: "SMALL", selected: "SMALL", options: 4},
		{templateID: "contact-form", key: "width", stored: "", selected: "FULL", options: 4},
		{templateID: "contact-form", key: "width", stored: "garbage", selected: "FULL", options: 4},

		{templateID: "folder", key: "format", stored: "TABLE", selected: "TABLE", options: 3},
		{templateID: "folder", key: "format", stored: "CARDS", selected: "CARDS", options: 3},
		{templateID: "folder", key: "format", stored: "COLUMNS", selected: "COLUMNS", options: 3},
		{templateID: "folder", key: "format", stored: "", selected: "TABLE", options: 3},
		{templateID: "folder", key: "format", stored: "garbage", selected: "TABLE", options: 3},
	}

	for _, test := range tests {

		template, exists := templateService.templatePrep[test.templateID]
		require.True(t, exists, "template %q not found", test.templateID)

		var buffer strings.Builder
		require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, "layout-controls", dataStub{test.key: test.stored}))
		output := buffer.String()

		require.Equal(t, test.options, strings.Count(output, "<option "), "%s stored %q", test.templateID, test.stored)
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
		{directory: "stream-article-two-column", field: "data.width"},
		{directory: "stream-article-two-column", field: "data.columns"},
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

// TestLayoutControls_TwoColumn covers the one Template whose layout slot carries TWO controls.
//
// The shared assertions above assume a single select, so this asserts the pair instead: the
// base's width control, reused unchanged, and the split control that article-two-column owns.
// Both must post, and exactly one option of each must be selected for every stored value --
// including none, which folds into ONE-HALF because an empty value renders "columns-", which
// matches no rule and leaves the stylesheet's equal-halves default in place.
//
// The split is DRAWN in the editor now, not here, so this slot is only two pickers.
func TestLayoutControls_TwoColumn(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	template, exists := templateService.templatePrep["article-two-column"]
	require.True(t, exists, "article-two-column not found")

	tests := []struct {
		width    string
		columns  string
		selected string // the column-split option expected to be selected
	}{
		{width: "FULL", columns: "TWO-THIRDS", selected: "TWO-THIRDS"},
		{width: "LARGE", columns: "ONE-HALF", selected: "ONE-HALF"},
		{width: "MEDIUM", columns: "ONE-THIRD", selected: "ONE-THIRD"},
		{width: "SMALL", columns: "TWO-THIRDS", selected: "TWO-THIRDS"},
		{width: "", columns: "", selected: "ONE-HALF"},
		{width: "garbage", columns: "garbage", selected: "ONE-HALF"},
	}

	for _, test := range tests {

		var buffer strings.Builder
		require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, "layout-controls",
			dataStub{"width": test.width, "columns": test.columns}))

		output := buffer.String()

		// Both settings must reach the POST, or one of them silently does nothing
		require.Contains(t, output, `name="data.width"`, "columns %q", test.columns)
		require.Contains(t, output, `name="data.columns"`, "columns %q", test.columns)

		// The width control is still a <select>; the split is the icon picker, which is radios
		// because an <option> cannot hold markup.  One selection apiece, counted separately.
		require.Equal(t, 4, strings.Count(output, "<option "), "columns %q", test.columns)
		require.Equal(t, 1, strings.Count(output, " selected>"), "columns %q", test.columns)
		require.Equal(t, 3, strings.Count(output, `<input type="radio"`), "columns %q", test.columns)
		require.Equal(t, 1, strings.Count(output, " checked>"), "columns %q", test.columns)
		require.Contains(t, output, `value="`+test.selected+`" checked>`, "columns %q", test.columns)

		// Only the width control may re-proportion the canvas, so the split control must not
		// wear its hook
		require.Contains(t, output, "widget-editor-width", "columns %q", test.columns)
		require.Contains(t, output, "two-column-split-picker", "columns %q", test.columns)
	}
}

// TestLayoutControls_CreateSeedsMatchWidthDefault asserts that a Template which seeds data.width
// when a Stream is created seeds the same value its schema declares as the default.
//
// The two are written in different files and nothing else connects them, so they drift in
// silence -- and the drift does not look like a bug, because both values stay inside the enum
// and every page still renders.  It has already happened once: the four article Templates seeded
// LARGE back when LARGE meant "the whole width", and the day a FULL option was added above it,
// every newly created article quietly started life at 75%.
//
// Templates that seed no width are skipped; they inherit the default and cannot disagree with it.
func TestLayoutControls_CreateSeedsMatchWidthDefault(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	checked := 0

	for templateID, template := range templateService.templatePrep {

		action, exists := template.Action("create")

		if !exists {
			continue
		}

		seed, exists := seededValue(action.Steps, "data.width")

		if !exists {
			continue
		}

		element, exists := template.Schema.GetElement("data.width")
		require.True(t, exists, "%s seeds data.width but declares no schema for it", templateID)

		stringElement, ok := element.(schema.String)
		require.True(t, ok, "%s: data.width must be a string", templateID)

		require.Equal(t, stringElement.Default, seed,
			"%s: the create action seeds data.width=%q while the schema defaults to %q",
			templateID, seed, stringElement.Default)

		checked++
	}

	require.Positive(t, checked, "no Template seeds data.width -- this test is watching nothing")
}

// seededValue returns the literal value a pipeline's set-data step writes to the named path,
// following any nested pipeline.  The second result is FALSE when no step seeds that path.
func seededValue(steps []step.Step, path string) (string, bool) {

	for _, item := range steps {

		if setData, ok := item.(step.SetData); ok {
			if value, exists := setData.Values[path]; exists && value.Tree != nil {
				return value.Tree.Root.String(), true
			}
		}

		if withDraft, ok := item.(step.WithDraft); ok {
			if value, exists := seededValue(withDraft.SubSteps, path); exists {
				return value, exists
			}
		}
	}

	return "", false
}

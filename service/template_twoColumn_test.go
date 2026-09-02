package service

import (
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// twoColumnStub stands in for the Stream builder, supplying just the accessors that the Two
// Column view and editor read.  Method names and signatures mirror build.Stream, so a rename
// there breaks this test instead of the page.
type twoColumnStub map[string]string

// Data returns a stubbed data value as an "any", mirroring build.Stream.Data -- nil included,
// because build.Stream.Data indexes a map[string]any and an unset layout value really is nil.
// html/template folds that to an empty string in an attribute, which is the whole reason an
// unset value can render "columns-" instead of a broken class name.
func (stub twoColumnStub) Data(key string) any {

	value, exists := stub[key]

	if !exists {
		return nil
	}

	return value
}

// DataString returns a stubbed data value, mirroring build.Stream.DataString
func (stub twoColumnStub) DataString(key string) string {
	return stub[key]
}

// StreamID returns a stubbed StreamID, mirroring build.Stream.StreamID
func (stub twoColumnStub) StreamID() string {
	return "000000000000000000000001"
}

// NavigationID returns a stubbed NavigationID, mirroring build.Stream.NavigationID
func (stub twoColumnStub) NavigationID() string {
	return "000000000000000000000002"
}

// Widgets returns no widgets, mirroring build.Stream.Widgets
func (stub twoColumnStub) Widgets(_ string) (template.HTML, error) {
	return template.HTML(""), nil
}

// UserCan denies every action, mirroring build.Stream.UserCan
func (stub twoColumnStub) UserCan(_ string) bool {
	return false
}

// loadTwoColumnTemplate returns the shipped article-two-column Template, after inheritance
func loadTwoColumnTemplate(t *testing.T) model.Template {
	t.Helper()

	templateService := loadEmbeddedTemplates(t)
	result, exists := templateService.templatePrep["article-two-column"]
	require.True(t, exists, "article-two-column not found")

	return result
}

// TestTwoColumn_ViewRendersBothColumns asserts that the page renders BOTH stored blocks, and
// renders them through the "markdown" helper.
//
// The helper is the only thing standing between data.left/data.right and the reader: both are
// stored with format "unsafe-any", which is raw Markdown source that nothing has inspected, so
// a view that printed either one directly would publish whatever the author pasted.  The
// script tag below is the check that the sanitizing actually happens.
func TestTwoColumn_ViewRendersBothColumns(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	stub := twoColumnStub{
		"left":    "# Left Heading\n\n<script>alert('left')</script>",
		"right":   "# Right Heading\n\n<script>alert('right')</script>",
		"width":   "MEDIUM",
		"columns": "TWO-THIRDS",
	}

	var buffer strings.Builder
	require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "view", stub))

	output := buffer.String()

	// Both columns are converted, not printed
	require.Contains(t, output, "<h1", "the left column must be converted from Markdown")
	require.Contains(t, output, "Left Heading")
	require.Contains(t, output, "Right Heading")

	// ...and sanitized on the way through
	require.NotContains(t, output, "<script>alert", "the markdown helper must sanitize both columns")

	// Both layout settings reach the page
	require.Contains(t, output, "layout-MEDIUM")
	require.Contains(t, output, "columns-TWO-THIRDS")

	// One e-content wrapper, so the h-entry still describes the whole article
	require.Equal(t, 1, strings.Count(output, "e-content"))
}

// TestTwoColumn_ViewUnsetColumns asserts that a Stream with no stored split still renders.
// An absent value writes "columns-", which matches no CSS rule and leaves the stylesheet's
// equal-halves default in place -- the same fold the Layout tab's final "else" makes.
func TestTwoColumn_ViewUnsetColumns(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	var buffer strings.Builder
	require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "view", twoColumnStub{"left": "hello"}))

	output := buffer.String()

	require.Contains(t, output, `class="e-content two-column columns-"`)
	require.Contains(t, output, "hello")
}

// TestTwoColumn_LayoutIsOneEngine asserts that the split is drawn by flexbox everywhere.
//
// Two surfaces draw the same layout -- the page and the editor -- and the editor's whole job is
// to show what the page will do.  Two engines means two sets of sizing rules that can drift
// apart silently, with the writing surface confidently showing a proportion the page never
// renders.  So: no grid in this stylesheet, and the same two child hooks in both.
//
// The Layout tab is not on this list.  Its slot picks the value; it no longer draws it.
//
// flex-basis:0 is the load-bearing line.  flex-grow shares out only the space left over after
// each item takes its basis, and the default basis is the item's own content -- so without it
// the ratios drift with whatever the author wrote, which bends the layout rather than breaking
// it, and nobody files a bug for that.
func TestTwoColumn_LayoutIsOneEngine(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	stylesheet, err := os.ReadFile("../_embed/templates/stream-article-two-column/stylesheet/two-column.css")
	require.NoError(t, err)

	css := string(stylesheet)

	require.NotContains(t, css, "display: grid", "the split must be drawn by ONE engine")
	require.NotContains(t, css, "grid-template-columns", "the split must be drawn by ONE engine")
	require.Contains(t, css, "flex-basis: 0", "flex-grow is only a true ratio when the basis is 0")

	// Every surface that draws the split wears the hooks the ratio rules select
	stub := twoColumnStub{"left": "left", "right": "right", "columns": "TWO-THIRDS"}

	for _, name := range []string{"view", "editor"} {

		var buffer strings.Builder
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, name, stub))

		output := buffer.String()

		require.Contains(t, output, "two-column-left", "%s must carry the ratio hooks", name)
		require.Contains(t, output, "two-column-right", "%s must carry the ratio hooks", name)
	}
}

// TestTwoColumn_ReflowsBelowMedium asserts that the columns reflow into rows below MEDIUM, and
// that they do it on the design system's own breakpoint and its own container.
//
// Every responsive utility in theme-global (.md\:flex-row, .cols-*, .md\:hide) is an
// @container query at 768px resolved against .page, which both shipped themes declare as an
// inline-size container.  Two things go wrong if this Template invents its own instead, and
// neither raises anything: a viewport @media query stops tracking the theme's chrome, and a
// container declared closer in -- on the article's own column -- makes the two width settings
// govern each other, so a layout-MEDIUM article can never split at all, because 66% of the
// widest page the theme draws is still under 768px.
func TestTwoColumn_ReflowsBelowMedium(t *testing.T) {

	stylesheet, err := os.ReadFile("../_embed/templates/stream-article-two-column/stylesheet/two-column.css")
	require.NoError(t, err)

	css := string(stylesheet)

	// Reflowed is the unconditional state, so every screen below the breakpoint gets it
	require.Contains(t, css, "flex-direction: column")

	// ...and MEDIUM is the one and only threshold
	conditions := regexp.MustCompile(`@container\s*\(([^)]*)\)`).FindAllStringSubmatch(css, -1)
	require.NotEmpty(t, conditions, "the split must reflow on a container query")

	for _, condition := range conditions {
		require.Equal(t, "min-width:768px", strings.ReplaceAll(condition[1], " ", ""),
			"SMALL and EXTRA-SMALL reflow, and nothing else is a breakpoint here")
	}

	require.NotContains(t, css, "@media screen",
		"a viewport query cannot see the theme's own chrome")
	require.NotContains(t, css, "container-type",
		"the query must resolve against .page, like every other responsive rule in the system")
}

// TestTwoColumn_EditorPostsBothColumns asserts that the editor page ships exactly the fields
// the "editor" action reads back, inside the form that posts them.
//
// The editors have to be INSIDE #saveForm: EasyMDE is built on CodeMirror.fromTextArea, which
// leaves the original textarea in place and writes back to it, so the form is what carries the
// two columns.  An editor outside the form would look identical and post nothing.
func TestTwoColumn_EditorPostsBothColumns(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	stub := twoColumnStub{
		"left":    "Left text",
		"right":   "Right text",
		"columns": "ONE-THIRD",
	}

	var buffer strings.Builder
	require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "editor", stub))

	output := buffer.String()

	// Every field the "editor" action reads back, seeded with what is stored
	require.Contains(t, output, `name="data.left"`)
	require.Contains(t, output, `name="data.right"`)
	require.Contains(t, output, "Left text")
	require.Contains(t, output, "Right text")

	// Nothing writes content.* for this Template, so nothing should post it
	require.NotContains(t, output, `name="content"`)

	// The editors are inside the form, which is what lets CodeMirror's own textareas post
	require.Contains(t, output, `id="saveForm"`)
	require.Less(t, strings.Index(output, `id="saveForm"`), strings.Index(output, `id="leftContent"`))

	// The picker posts the third field, and the writing surface takes its proportion from the
	// checked radio rather than from a stored class -- so a click re-proportions it at once
	require.Contains(t, output, `name="data.columns"`)
	require.NotContains(t, output, "columns-",
		"the editor row must carry no stored class: the checked radio is what drives it")
	require.Less(t,
		strings.Index(output, "two-column-split-picker"),
		strings.Index(output, "two-column-editor"),
		"the picker must precede the editor row: the ratio rules reach it with ~")
}

// TestTwoColumn_EditorPickerChecksExactlyOne asserts the Column Split picker for every value a
// Stream might have stored, including none.
//
// Three things have to hold at once and each fails silently on its own.  Exactly one radio is
// checked, so the field is present in every POST and the picker never renders blank.  An
// unrecognized or missing value folds to ONE-HALF, the same fold the Layout tab's final "else"
// makes.  And no option carries the literal text ZgotmplZ -- html/template writes that in place
// of any value interpolated where an attribute NAME belongs, the empty string included, so a
// conditional built by hoisting "selected" into a variable would put junk on every option that
// is not selected.  The inputs are emitted whole from inside their branches to avoid it.
func TestTwoColumn_EditorPickerChecksExactlyOne(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	tests := []struct {
		stored   string
		selected string
	}{
		{stored: "TWO-THIRDS", selected: "TWO-THIRDS"},
		{stored: "ONE-HALF", selected: "ONE-HALF"},
		{stored: "ONE-THIRD", selected: "ONE-THIRD"},
		{stored: "", selected: "ONE-HALF"},
		{stored: "garbage", selected: "ONE-HALF"},
	}

	for _, test := range tests {

		var buffer strings.Builder
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "editor",
			twoColumnStub{"left": "L", "right": "R", "columns": test.stored}))

		output := buffer.String()

		require.Equal(t, 3, strings.Count(output, `<input type="radio"`), "stored %q", test.stored)
		require.Equal(t, 1, strings.Count(output, " checked>"), "stored %q: exactly one radio", test.stored)
		require.Contains(t, output, `value="`+test.selected+`" checked>`, "stored %q", test.stored)
		require.NotContains(t, output, "ZgotmplZ", "stored %q", test.stored)

		// Each label must stay the input's NEXT sibling: every state the picker shows -- the
		// icon, the checked treatment, the focus ring -- is drawn with "input + label".
		//
		// Whitespace between them is fine and the template has some, because an adjacent-sibling
		// combinator skips text nodes.  Anything else in between is not, so this looks for an
		// input closed and a label opened with nothing but space between.
		labelFollowsInput := regexp.MustCompile(`<input type="radio"[^>]*>\s*<label for="columns`)
		require.Len(t, labelFollowsInput.FindAllString(output, -1), 3, "stored %q", test.stored)
	}
}

// TestTwoColumn_EditorActionSavesBothColumns asserts the save pipeline.
//
// The two columns and the split are ordinary custom fields, so set-data reads them and save
// writes them -- no edit-content, because there is no single content area for two blocks to
// occupy and view.html renders them straight from data.*.  The save step is the load-bearing
// half: set-data only mutates the draft in memory, so without it the POST succeeds and stores
// nothing.
//
// data.columns is read here as well as in the widgets action, because the editor ships a picker
// of its own.  Drop it from this list and the picker still moves, still saves, and still throws
// the choice away.
func TestTwoColumn_EditorActionSavesBothColumns(t *testing.T) {

	definition, err := os.ReadFile("../_embed/templates/stream-article-two-column/template.hjson")
	require.NoError(t, err)

	twoColumn := model.NewTemplate("stream-article-two-column", nil)
	require.NoError(t, hjson.Unmarshal(definition, &twoColumn))

	action, exists := twoColumn.Action("editor")
	require.True(t, exists, "article-two-column must define an editor action")

	withDraft, ok := action.Steps[0].(step.WithDraft)
	require.True(t, ok, "the editor action must run inside a draft")
	require.Len(t, withDraft.SubSteps, 2)

	setData, ok := withDraft.SubSteps[0].(step.SetData)
	require.True(t, ok, "set-data must read both columns out of the posted form")
	require.Equal(t, []string{"data.left", "data.right", "data.columns"}, setData.FromForm,
		"the split picker posts into this same form, so its field must be read back here too")

	_, ok = withDraft.SubSteps[1].(step.Save)
	require.True(t, ok, "set-data only mutates the draft in memory: save must follow it")
}

package service

import (
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/schema"
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
// flex-basis:0 is the important line.  flex-grow shares out only the space left over after
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

// twoColumnLadder is data.columns in ladder order, narrowest left column first.  Both surfaces
// list the radios this way, and the keyboard walks them in document order.
var twoColumnLadder = []string{"ONE-QUARTER", "ONE-THIRD", "ONE-HALF", "TWO-THIRDS", "THREE-QUARTERS"}

// TestTwoColumn_EditorPickerChecksExactlyOne asserts the Column Split control for every value a
// Stream might have stored, including none.  See the Template's AGENTS.md.
func TestTwoColumn_EditorPickerChecksExactlyOne(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	tests := []struct {
		stored   string
		selected string
	}{
		{stored: "THREE-QUARTERS", selected: "THREE-QUARTERS"},
		{stored: "TWO-THIRDS", selected: "TWO-THIRDS"},
		{stored: "ONE-HALF", selected: "ONE-HALF"},
		{stored: "ONE-THIRD", selected: "ONE-THIRD"},
		{stored: "ONE-QUARTER", selected: "ONE-QUARTER"},
		{stored: "", selected: "ONE-HALF"},
		{stored: "garbage", selected: "ONE-HALF"},
	}

	for _, test := range tests {

		var buffer strings.Builder
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "editor",
			twoColumnStub{"left": "L", "right": "R", "columns": test.stored}))

		output := buffer.String()

		// Exactly one radio is checked, so the field is present in every POST and the control
		// never renders blank.  An unrecognized or missing value folds to ONE-HALF.
		require.Equal(t, len(twoColumnLadder), strings.Count(output, `<input type="radio"`), "stored %q", test.stored)
		require.Equal(t, 1, strings.Count(output, " checked>"), "stored %q: exactly one radio", test.stored)
		checkedValue := regexp.MustCompile(`value="([A-Z-]+)"[^>]*checked>`).FindStringSubmatch(output)
		require.Len(t, checkedValue, 2, "stored %q: a radio must be checked", test.stored)
		require.Equal(t, test.selected, checkedValue[1], "stored %q", test.stored)

		// ZgotmplZ is what html/template writes in place of a value interpolated where an
		// attribute NAME belongs, so it appears the moment "checked" is hoisted into a variable
		require.NotContains(t, output, "ZgotmplZ", "stored %q", test.stored)
	}
}

// TestTwoColumn_EditorStopsAreWired asserts the markup contract the positioned options rest on.
//
// Every rule that places a stop, and every rule that gives it an icon, is written as
// "input[value=X] + label" -- so a label that stops being its input's next sibling loses its
// position and its picture at once, and lands unstyled on top of another option.
func TestTwoColumn_EditorStopsAreWired(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	var buffer strings.Builder
	require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, "editor",
		twoColumnStub{"left": "L", "right": "R", "columns": "ONE-HALF"}))

	output := buffer.String()

	// Whitespace between them is fine, because an adjacent-sibling combinator skips text nodes
	pairs := regexp.MustCompile(`<input type="radio"[^>]*>\s*<label for="columns`)
	require.Len(t, pairs.FindAllString(output, -1), len(twoColumnLadder),
		"every radio must be followed directly by its own label")

	// The variant class is what carries the positioning; without it the stops sit in a plain row
	require.Contains(t, output, "two-column-split-stops")

	// The radiogroup names itself, since the edit page shows no caption
	require.Contains(t, output, `role="radiogroup"`)
	require.Contains(t, output, `aria-label="Column widths"`)

	// The minifier escapes a "<" that does not open a tag, and a <script> body is raw text in
	// HTML -- so an escaped query literal reaches hyperscript as the entity and never parses
	hyperscript := regexp.MustCompile(`(?s)<script type="text/hyperscript">(.*?)</script>`).FindStringSubmatch(output)
	require.Len(t, hyperscript, 2, "the editor must ship its hyperscript block")
	require.NotContains(t, hyperscript[1], "&lt;", "a query literal was escaped on its way through the minifier")
}

// TestTwoColumn_LadderOrderOnBothSurfaces asserts that both places that set the split list it in
// the same direction, narrowest left column first.
//
// On the edit page that order is what the keyboard walks, and the stylesheet lays the options out
// left to right in the same sequence -- so a reordered list makes the arrow keys jump around the
// row instead of stepping along it. On the Layout tab it is what stops the icon row reading
// backwards, which is the original defect.
func TestTwoColumn_LadderOrderOnBothSurfaces(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	for _, name := range []string{"editor", "layout-controls"} {

		var buffer strings.Builder
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, name,
			twoColumnStub{"width": "FULL", "columns": "ONE-HALF"}))

		// Matched through the field name, so the width control's own options cannot be read as
		// rungs of this ladder
		found := regexp.MustCompile(`name="data\.columns" value="([A-Z-]+)"`).FindAllStringSubmatch(buffer.String(), -1)
		require.Len(t, found, len(twoColumnLadder), "%s: one radio per value", name)

		for index, match := range found {
			require.Equal(t, twoColumnLadder[index], match[1],
				"%s rung %d: narrowest left column first", name, index)
		}
	}
}

// TestTwoColumn_FocusRingHasATarget asserts that every class the focus ring is drawn on still
// exists in the markup of the surface it belongs to.
//
// The radios are invisible, so these rules are the ONLY indication of keyboard focus anywhere in
// this control. A ring left pointing at a deleted element takes focus off the screen entirely:
// the page renders, the control works by mouse, and a keyboard user is simply lost.
func TestTwoColumn_FocusRingHasATarget(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	stylesheet, err := os.ReadFile("../_embed/templates/stream-article-two-column/stylesheet/two-column.css")
	require.NoError(t, err)

	rule := regexp.MustCompile(`(?s)\n([^\n@}][^{]*):focus-visible([^{]*)\{[^}]*outline:[^}]*\}`).FindStringSubmatch(string(stylesheet))
	require.Len(t, rule, 3, "the control must draw a focus ring somewhere")

	var markup strings.Builder

	for _, name := range []string{"editor", "layout-controls"} {
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&markup, name,
			twoColumnStub{"width": "FULL", "columns": "ONE-HALF"}))
	}

	classes := regexp.MustCompile(`\.([a-zA-Z][\w-]*)`).FindAllStringSubmatch(rule[1]+rule[2], -1)
	require.NotEmpty(t, classes, "the focus rule must name the element it draws on")

	for _, class := range classes {
		require.Contains(t, markup.String(), class[1],
			"the focus ring is drawn on .%s, which nothing renders", class[1])
	}
}

// TestTwoColumn_EveryValueIsDrawn asserts that every value the schema admits is drawn on all
// three surfaces, and offered on both of the two that set it.
//
// A value that reaches the stylesheet nowhere is the quiet failure here: it validates, saves,
// and renders as equal halves, because "columns-WHATEVER" matches no rule.
func TestTwoColumn_EveryValueIsDrawn(t *testing.T) {

	twoColumn := loadTwoColumnTemplate(t)

	element, exists := twoColumn.Schema.GetElement("data.columns")
	require.True(t, exists, "article-two-column declares no schema for data.columns")

	stringElement, ok := element.(schema.String)
	require.True(t, ok, "data.columns must be a string")
	require.ElementsMatch(t, twoColumnLadder, stringElement.Enum,
		"the enum and the ladder must hold the same values")

	stylesheet, err := os.ReadFile("../_embed/templates/stream-article-two-column/stylesheet/two-column.css")
	require.NoError(t, err)

	css := string(stylesheet)

	surfaces := map[string]string{}

	for _, name := range []string{"editor", "layout-controls"} {

		var buffer strings.Builder
		require.NoError(t, twoColumn.HTMLTemplate.ExecuteTemplate(&buffer, name,
			twoColumnStub{"width": "FULL", "columns": "ONE-HALF"}))

		surfaces[name] = buffer.String()
	}

	for _, value := range stringElement.Enum {

		// Both places that set the field offer every value, or one of them cannot reach it
		for name, output := range surfaces {
			require.Contains(t, output, `value="`+value+`"`, "%s omits %s", name, value)
		}

		// The icon each surface paints it with, and the file that icon comes from
		require.Contains(t, css, `input[value="`+value+`"] + label::before`, "no Layout tab icon for %s", value)

		_, err := os.Stat("../_embed/templates/stream-article-two-column/resources/columns-" + value + ".svg")
		require.NoError(t, err, "no icon file for %s", value)

		// ONE-HALF is drawn by the equal-halves default, so it has no ratio rule of its own
		if value == "ONE-HALF" {
			continue
		}

		require.Contains(t, css, ".two-column.columns-"+value+" > .two-column-", "the page never draws %s", value)
		require.Contains(t, css, `:has([value="`+value+`"]:checked) ~ .two-column-editor > .two-column-`, "the editor never draws %s", value)
		require.Contains(t, css, `.two-column-split-stops input[value="`+value+`"] + label`,
			"%s has no stop of its own, so it lands on top of another option", value)
	}
}

// TestTwoColumn_EditorActionSavesBothColumns asserts the save pipeline.
//
// The two columns and the split are ordinary custom fields, so set-data reads them and save
// writes them -- no edit-content, because there is no single content area for two blocks to
// occupy and view.html renders them straight from data.*.  The save step is the important
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

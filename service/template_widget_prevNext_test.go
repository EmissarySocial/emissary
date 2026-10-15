package service

import (
	"html/template"
	"io/fs"
	"os"
	"strings"
	"testing"

	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
	formwidget "github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Previous/Next Buttons Widget
 *
 * These tests execute the real templates in
 * _embed/templates/widget-prev-next against a
 * stand-in for the build.Widget context.
 ******************************************/

// prevNextFakeStream stands in for a sibling build.Stream, or an empty one when absent
type prevNextFakeStream struct {
	absent  bool
	token   string
	label   string
	summary string
	icon    string
}

// IsEmpty returns TRUE when there is no sibling, matching build.Stream
func (stream prevNextFakeStream) IsEmpty() bool { return stream.absent }

// Token returns the sibling's URL token
func (stream prevNextFakeStream) Token() string { return stream.token }

// Label returns the sibling's title
func (stream prevNextFakeStream) Label() string { return stream.label }

// Summary returns the sibling's summary
func (stream prevNextFakeStream) Summary() string { return stream.summary }

// Icon returns the sibling's icon name
func (stream prevNextFakeStream) Icon() string { return stream.icon }

// prevNextFakeBuilder stands in for build.Widget: the widget's settings and the Stream's relatives
type prevNextFakeBuilder struct {
	Widget     navFakeWidget
	previous   prevNextFakeStream
	next       prevNextFakeStream
	parent     prevNextFakeStream
	firstChild prevNextFakeStream
	sorts      *[]string
}

// HasParent returns TRUE when the Stream has a parent, matching build.Stream
func (builder prevNextFakeBuilder) HasParent() bool {
	return !builder.parent.absent
}

// Parent returns the parent, as build.Stream.Parent does
func (builder prevNextFakeBuilder) Parent(_ string) (prevNextFakeStream, error) {
	return builder.parent, nil
}

// FirstChild returns the first child, and records the sort field it was asked for
func (builder prevNextFakeBuilder) FirstChild(sortField string, _ string) (prevNextFakeStream, error) {
	*builder.sorts = append(*builder.sorts, sortField)
	return builder.firstChild, nil
}

// PrevSibling returns the previous sibling, and records the sort field it was asked for
func (builder prevNextFakeBuilder) PrevSibling(sortField string, _ string) (prevNextFakeStream, error) {
	*builder.sorts = append(*builder.sorts, sortField)
	return builder.previous, nil
}

// NextSibling returns the next sibling, and records the sort field it was asked for
func (builder prevNextFakeBuilder) NextSibling(sortField string, _ string) (prevNextFakeStream, error) {
	*builder.sorts = append(*builder.sorts, sortField)
	return builder.next, nil
}

// renderPrevNextWidget parses the real widget-prev-next folder the way the server does, then
// renders it, returning the output and the sort fields the template asked for.
func renderPrevNextWidget(t *testing.T, settings mapof.Any, previous prevNextFakeStream, next prevNextFakeStream) (string, []string) {
	t.Helper()
	return renderPrevNextFamily(t, settings, previous, next, prevNextNone, prevNextNone)
}

// renderPrevNextFamily renders the widget for a Stream with the given siblings, parent, and first child.
func renderPrevNextFamily(t *testing.T, settings mapof.Any, previous prevNextFakeStream, next prevNextFakeStream, parent prevNextFakeStream, firstChild prevNextFakeStream) (string, []string) {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	widgetTemplate := template.New("widget-prev-next").Funcs(funcMap)
	require.NoError(t, loadHTMLTemplateFromFilesystem(os.DirFS("../_embed/templates/widget-prev-next"), widgetTemplate, funcMap))

	sorts := []string{}
	context := prevNextFakeBuilder{Widget: navFakeWidget{Data: settings}, previous: previous, next: next, parent: parent, firstChild: firstChild, sorts: &sorts}

	var buffer strings.Builder
	require.NoError(t, widgetTemplate.ExecuteTemplate(&buffer, "widget", context))

	return buffer.String(), sorts
}

// prevNextNone stands for a relative that does not exist
var prevNextNone = prevNextFakeStream{absent: true}

// prevNextParent and prevNextChild are a parent and a first child, for the family fallback
var prevNextParent = prevNextFakeStream{token: "docs", label: "Docs"}
var prevNextChild = prevNextFakeStream{token: "chapter-1", label: "Chapter 1"}

// prevNextAbout and prevNextContact are two siblings with every optional field filled
var prevNextAbout = prevNextFakeStream{token: "about", label: "About Me", summary: "Who I am", icon: "person"}
var prevNextContact = prevNextFakeStream{token: "contact", label: "Contact Me", summary: "Say hello", icon: "envelope"}

// TestPrevNextWidget_BothSiblings renders both buttons with titles, the default settings.
func TestPrevNextWidget_BothSiblings(t *testing.T) {

	result, sorts := renderPrevNextWidget(t, mapof.Any{}, prevNextAbout, prevNextContact)

	require.Equal(t, []string{"rank", "rank"}, sorts, "an unset sort reads siblings in rank order")
	require.Contains(t, result, `prev-next-link previous sibling`)
	require.Contains(t, result, `prev-next-link next sibling`)
	require.Contains(t, result, `<a href="/about" hx-get="/about" rel="prev" class="turboclick prev-next-link previous sibling">`)
	require.Contains(t, result, `<a href="/contact" hx-get="/contact" rel="next" class="turboclick prev-next-link next sibling">`)
	require.Contains(t, result, `<span class="prev-next-title">About Me</span>`)
	require.Contains(t, result, `<span class="prev-next-title">Contact Me</span>`)
	require.Less(t, strings.Index(result, "/about"), strings.Index(result, "/contact"), "previous comes first")
	require.NotContains(t, result, "prev-next-summary")
	require.NotContains(t, result, "prev-next-icon")
	require.NotContains(t, result, "<nav", "a <nav> would pick up the site bar's styles")
}

// TestPrevNextWidget_AllOptions renders titles, summaries, and icons, with a chosen sort.
func TestPrevNextWidget_AllOptions(t *testing.T) {

	result, sorts := renderPrevNextWidget(t, mapof.Any{"sort": "label", "showSummary": true, "showIcon": true}, prevNextAbout, prevNextContact)

	require.Equal(t, []string{"label", "label"}, sorts)
	require.Contains(t, result, `<span class="prev-next-summary">Who I am</span>`)
	require.Contains(t, result, `<span class="prev-next-summary">Say hello</span>`)
	require.Equal(t, 2, strings.Count(result, `<span class="prev-next-icon">`))
}

// TestPrevNextWidget_LabelsOffByDefault leaves out the direction labels for a widget that never set them.
func TestPrevNextWidget_LabelsOffByDefault(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{}, prevNextAbout, prevNextContact)

	require.NotContains(t, result, "prev-next-direction")
	require.Contains(t, result, `<span class="prev-next-title">About Me</span>`)
	require.Contains(t, result, `<span class="prev-next-title">Contact Me</span>`)
}

// TestPrevNextWidget_ShowLabels adds a direction label above each title.
func TestPrevNextWidget_ShowLabels(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{"showLabels": true}, prevNextAbout, prevNextContact)

	require.Contains(t, result, `<span class="prev-next-direction">Previous</span>`)
	require.Contains(t, result, `<span class="prev-next-direction">Next</span>`)
	require.Less(t, strings.Index(result, "Previous"), strings.Index(result, "About Me"), "the label sits above the title")
	require.Less(t, strings.Index(result, ">Next<"), strings.Index(result, "Contact Me"), "the label sits above the title")
}

// TestPrevNextWidget_TitlesAlwaysShow ignores the retired showTitle setting saved by older widgets.
func TestPrevNextWidget_TitlesAlwaysShow(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{"showTitle": false}, prevNextAbout, prevNextContact)

	require.Contains(t, result, `<span class="prev-next-title">About Me</span>`)
	require.Contains(t, result, `<span class="prev-next-title">Contact Me</span>`)
}

// TestPrevNextWidget_MissingOptionalFields omits a summary or icon line the sibling does not have.
func TestPrevNextWidget_MissingOptionalFields(t *testing.T) {

	bare := prevNextFakeStream{token: "bare", label: "Bare"}
	result, _ := renderPrevNextWidget(t, mapof.Any{"showSummary": true, "showIcon": true}, bare, prevNextFakeStream{absent: true})

	require.Contains(t, result, `<span class="prev-next-title">Bare</span>`)
	require.NotContains(t, result, "prev-next-summary")
	require.NotContains(t, result, "prev-next-icon")
}

// TestPrevNextWidget_OneSide renders a single button, with its own side's class.
func TestPrevNextWidget_OneSide(t *testing.T) {

	onlyNext, _ := renderPrevNextWidget(t, mapof.Any{"includeFamily": false}, prevNextFakeStream{absent: true}, prevNextContact)
	require.NotContains(t, onlyNext, `rel="prev"`)
	require.Contains(t, onlyNext, `prev-next-link next`)

	onlyPrevious, _ := renderPrevNextWidget(t, mapof.Any{"includeFamily": false}, prevNextAbout, prevNextFakeStream{absent: true})
	require.Contains(t, onlyPrevious, `prev-next-link previous`)
	require.NotContains(t, onlyPrevious, `rel="next"`)
}

// TestPrevNextWidget_NoSiblings renders nothing at all for a Stream with no neighbors.
func TestPrevNextWidget_NoSiblings(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{}, prevNextFakeStream{absent: true}, prevNextFakeStream{absent: true})

	require.Equal(t, "", strings.TrimSpace(result))
}

// TestPrevNextWidget_EscapesLabels confirms that sibling text is escaped, not rendered as markup.
func TestPrevNextWidget_EscapesLabels(t *testing.T) {

	hostile := prevNextFakeStream{token: "x", label: `<img src=x onerror=alert(1)>`, summary: `<script>alert(2)</script>`}
	result, _ := renderPrevNextWidget(t, mapof.Any{"showSummary": true}, hostile, prevNextFakeStream{absent: true})

	require.NotContains(t, result, "<img")
	require.NotContains(t, result, "<script")
	require.Contains(t, result, "&lt;img src=x onerror=alert(1)&gt;")
}

// TestPrevNextWidget_FamilyOnByDefault falls back to the parent and the first child when the
// option was never saved, which is how every widget placed before the option existed reads.
func TestPrevNextWidget_FamilyOnByDefault(t *testing.T) {

	result, sorts := renderPrevNextFamily(t, mapof.Any{"sort": "label"}, prevNextNone, prevNextNone, prevNextParent, prevNextChild)

	require.Contains(t, result, `<a href="/docs" hx-get="/docs" rel="prev" class="turboclick prev-next-link previous parent">`)
	require.Contains(t, result, `<a href="/chapter-1" hx-get="/chapter-1" rel="next" class="turboclick prev-next-link next child">`)
	require.Equal(t, []string{"label", "label", "label"}, sorts, "the first child is found with the same sort")
}

// TestPrevNextWidget_SiblingsWinOverFamily keeps siblings when they exist.
func TestPrevNextWidget_SiblingsWinOverFamily(t *testing.T) {

	result, _ := renderPrevNextFamily(t, mapof.Any{"includeFamily": true}, prevNextAbout, prevNextContact, prevNextParent, prevNextChild)

	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)
	require.NotContains(t, result, `href="/docs"`)
	require.NotContains(t, result, `href="/chapter-1"`)
}

// TestPrevNextWidget_FamilyOff ignores the parent and children when the option is turned off.
func TestPrevNextWidget_FamilyOff(t *testing.T) {

	result, _ := renderPrevNextFamily(t, mapof.Any{"includeFamily": false}, prevNextNone, prevNextNone, prevNextParent, prevNextChild)

	require.Equal(t, "", strings.TrimSpace(result))
}

// TestPrevNextWidget_TopLevelNoParent never asks for the parent of a Stream that has none.
func TestPrevNextWidget_TopLevelNoParent(t *testing.T) {

	result, _ := renderPrevNextFamily(t, mapof.Any{}, prevNextNone, prevNextContact, prevNextNone, prevNextNone)

	require.NotContains(t, result, `rel="prev"`)
	require.Contains(t, result, `href="/contact"`)
}

// TestPrevNextWidget_ArrowsBesideTitles puts the arrow before the previous title and after the
// next title, and keeps the direction labels free of arrows.
func TestPrevNextWidget_ArrowsBesideTitles(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{"showLabels": true}, prevNextAbout, prevNextContact)

	require.Contains(t, result, `<span class="prev-next-direction">Previous</span>`)
	require.Contains(t, result, `<span class="prev-next-direction">Next</span>`)

	previousArrow := strings.Index(result, `&#8249;`)
	previousTitle := strings.Index(result, `About Me`)
	nextTitle := strings.Index(result, `Contact Me`)
	nextArrow := strings.Index(result, `&#8250;`)

	require.Less(t, previousArrow, previousTitle, "the previous arrow leads its title")
	require.Less(t, nextTitle, nextArrow, "the next arrow trails its title")
	require.Equal(t, 2, strings.Count(result, `<span class="prev-next-arrow" aria-hidden="true">`))
}

// TestPrevNextWidget_ArrowsWithoutLabels keeps both arrows when the direction labels are off.
func TestPrevNextWidget_ArrowsWithoutLabels(t *testing.T) {

	result, _ := renderPrevNextWidget(t, mapof.Any{}, prevNextAbout, prevNextContact)

	require.Equal(t, 2, strings.Count(result, `class="prev-next-arrow"`))
	require.NotContains(t, result, "prev-next-direction")
}

// renderPrevNextEditor loads the widget definition and renders its editor form for data,
// as the widget editor does.
func renderPrevNextEditor(t *testing.T, data mapof.Any) string {

	t.Helper()
	formwidget.UseAll()

	filesystem := os.DirFS("../_embed/templates/widget-prev-next")
	definition, err := fs.ReadFile(filesystem, "widget.hjson")
	require.NoError(t, err)

	widgetService := NewWidget(emissarytemplates.FuncMap(nullIconProvider{}))
	require.NoError(t, widgetService.Add("prev-next", filesystem, definition))

	widget, ok := widgetService.Get("prev-next")
	require.True(t, ok)

	result, err := form.Editor(widget.Schema, widget.Form, data, nil)
	require.NoError(t, err)
	return result
}

// TestPrevNextWidget_EditorSortIsLastAndRadio puts "Order Siblings By" last, as radio buttons.
func TestPrevNextWidget_EditorSortIsLastAndRadio(t *testing.T) {

	result := renderPrevNextEditor(t, mapof.Any{"sort": "label"})

	require.NotContains(t, result, "<select", "the sort order is not a select box")
	require.Equal(t, 3, strings.Count(result, `type="radio"`))
	require.Greater(t, strings.Index(result, "Order Siblings By"), strings.Index(result, "Include Parent and Children"), "the sort order is the last option")
	require.Regexp(t, `value="label"[^>]*checked`, result, "a saved sort is selected")
}

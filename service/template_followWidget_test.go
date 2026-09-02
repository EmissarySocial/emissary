package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"net/url"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"

	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// followWidgetStub stands in for the Widget builder, supplying just the accessors that the
// Follow Button reads.  Field and method names mirror build.Widget (and the *build.Stream it
// embeds), so a rename there breaks this test instead of the page.
type followWidgetStub struct {
	Widget  *model.StreamWidget
	canEdit bool
}

// UserCan reports the stubbed edit permission, mirroring build.Stream.UserCan
func (stub followWidgetStub) UserCan(_ string) bool {
	return stub.canEdit
}

// newFollowWidgetStub returns a stub carrying the provided widget data
func newFollowWidgetStub(data mapof.Any, canEdit bool) followWidgetStub {
	return followWidgetStub{
		Widget:  &model.StreamWidget{Data: data},
		canEdit: canEdit,
	}
}

// loadFollowWidget returns the shipped "follow" Widget, loaded through the real service path so
// that the hjson definition, its schema formats, and the minify-then-parse of widget.html are
// all exercised exactly as they are at startup.
func loadFollowWidget(t *testing.T) model.Widget {
	t.Helper()

	const directory = "widget-follow"

	widgetService := NewWidget(emissarytemplates.FuncMap(nullIconProvider{}))

	filesystem := os.DirFS(filepath.Join("../_embed/templates", directory))
	definitionType, definition := findDefinition(filesystem)
	require.Equal(t, DefinitionWidget, definitionType, "%s must contain a widget definition", directory)
	require.NoError(t, widgetService.Add(directory, filesystem, definition))

	result, exists := widgetService.Get("follow")
	require.True(t, exists, "the widget must register under the id declared in widget.hjson")

	return result
}

// renderFollowWidget executes the Follow Button's "widget" template against a stub
func renderFollowWidget(t *testing.T, widget model.Widget, stub followWidgetStub) string {
	t.Helper()

	var buffer strings.Builder
	require.NoError(t, widget.HTMLTemplate.ExecuteTemplate(&buffer, "widget", stub))

	return buffer.String()
}

// TestFollowWidget_Renders asserts that a configured widget produces a button addressed at the
// user-outbox "follow" action, with the two htmx attributes that as-modal does not supply itself.
//
// The button must NOT carry an hx-target of its own.  That action is wrapped in as-modal, which
// answers with HX-Retarget:aside and HX-Reswap:innerHTML; a local hx-target here would look
// harmless and would be overridden anyway, but it is the kind of thing that gets copied into the
// next widget, where nothing overrides it.
//
// It MUST carry hx-push-url="false", for the opposite reason: as-modal sends HX-Push-Url:false
// only when the action declares no background, and "follow" declares background:"profile".  This
// widget is the one caller of that action from a page that is not the profile, so without the
// attribute the address bar walks away from whatever the visitor was actually reading.
func TestFollowWidget_Renders(t *testing.T) {

	widget := loadFollowWidget(t)

	stub := newFollowWidgetStub(mapof.Any{"username": "benpate", "label": "Follow Me"}, false)
	output := renderFollowWidget(t, widget, stub)

	require.Contains(t, output, `hx-get="/@benpate/follow"`)
	require.Contains(t, output, "Follow Me")
	require.Contains(t, output, `class="primary"`)
	require.Contains(t, output, `hx-push-url="false"`)
	require.NotContains(t, output, "hx-target")
}

// TestFollowWidget_DefaultLabel asserts that an empty label falls back to the word "Follow",
// rather than rendering a button with no text on it at all.
func TestFollowWidget_DefaultLabel(t *testing.T) {

	widget := loadFollowWidget(t)

	stub := newFollowWidgetStub(mapof.Any{"username": "benpate"}, false)
	output := renderFollowWidget(t, widget, stub)

	require.Contains(t, output, `hx-get="/@benpate/follow"`)
	require.Contains(t, output, ">Follow<")
}

// TestFollowWidget_NoUsername asserts what an unconfigured widget does: it explains itself to
// somebody who can fix it, and renders nothing at all for everybody else.
//
// The second half is the one that matters.  A widget dropped onto a page and saved before it was
// filled in is a normal thing for an author to do, and the resulting page is public; a visitor
// must not be shown the site's configuration mistakes.  Neither case may emit a button, because
// "/@/follow" is not a route.
func TestFollowWidget_NoUsername(t *testing.T) {

	widget := loadFollowWidget(t)

	// An author sees the explanation...
	author := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"label": "Follow Me"}, true))
	require.Contains(t, author, "no username has been set")
	require.NotContains(t, author, "hx-get")

	// ...and a visitor sees nothing whatsoever
	visitor := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"label": "Follow Me"}, false))
	require.Equal(t, "", strings.TrimSpace(visitor))
}

// TestFollowWidget_CustomClassAndStyle asserts that both author-supplied styling hooks land on
// the widget's own element, alongside its built-in classes rather than replacing them.
func TestFollowWidget_CustomClassAndStyle(t *testing.T) {

	widget := loadFollowWidget(t)

	stub := newFollowWidgetStub(mapof.Any{
		"username": "benpate",
		"class":    "my-cta padding-lg",
		"style":    "margin-top: 1rem; color: red;",
	}, false)

	output := renderFollowWidget(t, widget, stub)

	require.Contains(t, output, `class="widget widget-follow my-cta padding-lg"`)
	require.Contains(t, output, `style="margin-top: 1rem; color: red;"`)

	// The widget's own hooks survive when the author sets neither
	bare := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"username": "benpate"}, false))
	require.Contains(t, bare, "widget widget-follow")

	// The unconfigured note is a diagnostic, not the widget, so the author's styling is not
	// applied to it -- a class or style that positions or hides the button must not hide this.
	note := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"class": "hidden", "style": "display:none"}, true))
	require.Contains(t, note, "no username has been set")
	require.NotContains(t, note, "hidden")
	require.NotContains(t, note, "display:none")
}

// TestFollowWidget_StyleAttributeIsNotZgotmplZ pins the one thing about this field that fails
// silently and completely.
//
// html/template reads a `style` attribute as CSS context and runs its own value filter there.  A
// plain string holding a declaration list does not survive that filter: it is replaced wholesale
// with the literal text "ZgotmplZ", so the attribute is not merely wrong, it never works at all --
// and nothing errors.  Piping through `css` is what suppresses the filter, and this test is what
// notices if that pipe is ever dropped.
func TestFollowWidget_StyleAttributeIsNotZgotmplZ(t *testing.T) {

	widget := loadFollowWidget(t)

	stub := newFollowWidgetStub(mapof.Any{"username": "benpate", "style": "margin-top: 1rem; color: red;"}, false)
	output := renderFollowWidget(t, widget, stub)

	require.NotContains(t, output, "ZgotmplZ")
	require.Contains(t, output, `style="margin-top: 1rem; color: red;"`)
}

// TestFollowWidget_ClassFormatBoundsTheAttribute asserts that the schema confines the class to
// characters that can only ever be class names.
//
// html/template escapes this value in attribute context, so a quote here cannot break out of the
// attribute on its own; the schema is what stops a styling hook from being used to park arbitrary
// text in the page.  Whitespace and non-Latin letters must survive, because a class attribute
// holding several classes is the normal case and class names are not required to be English.
func TestFollowWidget_ClassFormatBoundsTheAttribute(t *testing.T) {

	widget := loadFollowWidget(t)

	rejected := []string{
		`evil" onclick="x`,
		"a;b",
		"a<b",
		"a>b",
		"a&b",
		"url(javascript:alert(1))",
	}

	for _, class := range rejected {
		_, err := widget.Schema.Validate(mapof.Any{"class": class})
		require.Error(t, err, "class %q must be rejected by the schema", class)
	}

	accepted := []string{
		"card",
		"card padding-lg",
		"card ",
		"kärnan",
		"",
	}

	for _, class := range accepted {
		_, err := widget.Schema.Validate(mapof.Any{"class": class})
		require.NoError(t, err, "class %q must be accepted by the schema", class)
	}
}

// saveFollowWidget runs values through the widget's real save path -- the same form.SetURLValues
// call StepEditWidget.Post makes -- and returns the data as it would be stored.
//
// This is deliberately NOT schema.Validate.  Validate answers "does this value already conform",
// so it reports an error for anything a format had to rewrite; every sanitizing format therefore
// "fails" it even when it did its job.  Set is the call that rewrites in place, and it is the one
// the save path actually makes, so it is the one worth asserting against.
func saveFollowWidget(t *testing.T, followWidget model.Widget, values url.Values) mapof.Any {
	t.Helper()

	// SetURLValues resolves each element's Type through the form registry, which is populated
	// by UseAll rather than by an init().  Without it every element -- not just the new ones --
	// fails with "Unable to locate widget for element".
	widget.UseAll()

	data := mapof.NewAny()
	f := form.New(followWidget.Schema, followWidget.Form)
	require.NoError(t, f.SetURLValues(&data, values, nil))

	return data
}

// TestFollowWidget_StyleFormatSanitizes asserts that the style field is confined to the same
// policy Emissary already applies to an author's page stylesheet.
//
// This field is written into a `style` attribute through a trust cast, so the schema format is the
// only thing deciding what an author may put there.  The two entries that matter are the ones the
// allowlist exists for: `position:fixed` would let a widget lift itself out of the flow and cover
// unrelated page chrome, and any `url()` would make the page fetch a third-party resource on every
// view.  Both are dropped here exactly as they are dropped from a stylesheet.
//
// The format sanitizes rather than rejects, so an unsafe declaration is stripped and the rest is
// kept.  These are assertions about what gets stored, not about whether the save succeeded.
func TestFollowWidget_StyleFormatSanitizes(t *testing.T) {

	widget := loadFollowWidget(t)

	tests := []struct {
		name     string
		style    string
		expected string
	}{
		{"ordinary declarations survive", "margin-top:1rem; color:red", "margin-top: 1rem; color: red;"},
		{"overlay positioning is dropped", "position:fixed; top:0; color:red", "color: red;"},
		{"in-flow positioning survives", "position:relative; color:red", "position: relative; color: red;"},
		{"resource loading is dropped", "background:url(https://example.com/beacon); color:red", "color: red;"},
		{"legacy script hooks are dropped", "behavior:url(x.htc); color:red", "color: red;"},
		{"a quote discards the whole value", `color:red" onmouseover="alert(1)`, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := saveFollowWidget(t, widget, url.Values{"style": {test.style}})
			require.Equal(t, test.expected, data.GetString("style"))
		})
	}
}

// TestFollowWidget_SaveRoundTrip asserts that a whole settings form saves the way an author filled
// it in, and that what comes back out is what the page renders.
func TestFollowWidget_SaveRoundTrip(t *testing.T) {

	widget := loadFollowWidget(t)

	data := saveFollowWidget(t, widget, url.Values{
		"username": {"benpate"},
		"label":    {"Follow Me"},
		"class":    {"my-cta"},
		"style":    {"margin-top:1rem"},
	})

	require.Equal(t, "benpate", data.GetString("username"))
	require.Equal(t, "Follow Me", data.GetString("label"))
	require.Equal(t, "my-cta", data.GetString("class"))
	require.Equal(t, "margin-top: 1rem;", data.GetString("style"))

	output := renderFollowWidget(t, widget, followWidgetStub{Widget: &model.StreamWidget{Data: data}})

	require.Contains(t, output, `class="widget widget-follow my-cta"`)
	require.Contains(t, output, `style="margin-top: 1rem;"`)
	require.Contains(t, output, ">Follow Me<")
}

// TestFollowWidget_FormTabs asserts the shape of the settings form: two tabs, named, with each
// field on the one it belongs to.
//
// LayoutTabs takes each tab's name from its child element's Label, so a tab whose layout-vertical
// has no label silently renders as "Tab 0".
func TestFollowWidget_FormTabs(t *testing.T) {

	widget := loadFollowWidget(t)

	require.Equal(t, "layout-tabs", widget.Form.Type)
	require.Len(t, widget.Form.Children, 2)

	require.Equal(t, "Button", widget.Form.Children[0].Label)
	require.Equal(t, "Stylesheet", widget.Form.Children[1].Label)

	require.Equal(t, []string{"username", "label"}, formPaths(widget.Form.Children[0]))
	require.Equal(t, []string{"class", "style"}, formPaths(widget.Form.Children[1]))
}

// formPaths returns the paths edited by a form Element's immediate children
func formPaths(element form.Element) []string {

	result := make([]string, 0, len(element.Children))

	for _, child := range element.Children {
		result = append(result, child.Path)
	}

	return result
}

// TestFollowWidget_UsernameValidator asserts that the username field asks the server whether the
// name it was given belongs to anybody, and that the field it names is the one the handler accepts.
//
// The two ends of this agree only by convention.  The validator behavior sends "field=" plus the
// input's name attribute, which the form library renders from the element's Path; every handler in
// handler/validate.go answers HTTP 400 for any other field name.  So renaming this path silently
// turns the validator off -- the fetch keeps succeeding, and its 400 body carries valid:false
// forever, marking every username wrong.
//
// The URL itself is checked against the real route table by TestValidatorURLsAreRouted.
func TestFollowWidget_UsernameValidator(t *testing.T) {

	widget := loadFollowWidget(t)

	username, exists := findFormElement(widget.Form, "username")
	require.True(t, exists, "the form must offer a username field")
	require.Equal(t, "/.validate/user/exists", username.Options.GetString("validator"))

	// The handler keys off the field name, and the field name is this path
	require.Equal(t, "username", username.Path)
}

// findFormElement returns the descendant of a form Element that edits the provided path
func findFormElement(element form.Element, path string) (form.Element, bool) {

	if element.Path == path {
		return element, true
	}

	for _, child := range element.Children {
		if result, exists := findFormElement(child, path); exists {
			return result, true
		}
	}

	return form.Element{}, false
}

// TestFollowWidget_UsernameFormatGuardsTheURL asserts that the schema is what keeps the username
// safe to write straight into the hx-get URL in widget.html.
//
// The rendered value is not escaped for a URL at the render site -- it is escaped for HTML, which
// would happily pass through a "/" or a "?" and let an author aim the button at a different route
// entirely.  The "username" format is the only thing standing there, so if this test starts
// failing because the format was relaxed, widget.html needs escaping before the schema changes.
func TestFollowWidget_UsernameFormatGuardsTheURL(t *testing.T) {

	widget := loadFollowWidget(t)

	tests := []string{
		"benpate/../admin",
		"benpate?intent=follow",
		"benpate follow",
		`benpate" hx-target="body`,
		"benpate#fragment",
	}

	for _, username := range tests {
		value := mapof.Any{"username": username}
		_, err := widget.Schema.Validate(value)
		require.Error(t, err, "username %q must be rejected by the schema", username)
	}

	// ...and an ordinary username still passes
	_, err := widget.Schema.Validate(mapof.Any{"username": "ben_pate99", "label": "Follow Me"})
	require.NoError(t, err)
}

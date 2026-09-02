package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
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

// TestFollowWidget_CustomClass asserts that the author's class lands on a wrapper of its own,
// outside the widget's own element.
//
// Keeping the two apart is the point.  If the author's class were merged onto .widget-follow, a
// page stylesheet and the widget's own hooks would be selecting the same element, and either could
// silently win over the other; a separate wrapper gives the author a box to position without
// reaching inside the widget.  The wrapper is unconditional, so the DOM shape does not change when
// the class is cleared and a rule written against the structure keeps matching.
func TestFollowWidget_CustomClass(t *testing.T) {

	widget := loadFollowWidget(t)

	stub := newFollowWidgetStub(mapof.Any{"username": "benpate", "class": "my-cta padding-lg"}, false)
	output := renderFollowWidget(t, widget, stub)

	require.Contains(t, output, `<div class="my-cta padding-lg">`)
	require.Contains(t, output, `<div class="widget widget-follow">`)

	// The wrapper is still there, empty, when no class is set
	bare := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"username": "benpate"}, false))
	require.Contains(t, bare, `<div class="">`)

	// The unconfigured note is a diagnostic, not the widget, so it is not wrapped or styled by
	// the author -- a class that positions or hides the button must not be able to hide this.
	note := renderFollowWidget(t, widget, newFollowWidgetStub(mapof.Any{"class": "hidden"}, true))
	require.Contains(t, note, "no username has been set")
	require.NotContains(t, note, "hidden")
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

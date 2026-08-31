package service

import (
	"html/template"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Minimal Theme: Navigation Toggles
 *
 * The "minimal" theme lets a Domain owner hide the top-level
 * navigation from the theme settings form.  These tests run the
 * REAL theme.hjson and the REAL navigation.html, because the two
 * halves are only correct together: the form writes a boolean
 * into Domain.ThemeData, and the template reads it back through
 * .ThemeData, which stringifies it.
 *
 * The trap they exist to pin down is the DEFAULT.  Domain.ThemeData
 * starts out empty, an absent boolean reads as FALSE, and the theme
 * form posts every tab at once -- so without the "default" declared
 * on these properties, the toggles would render OFF and the first
 * save of ANY tab would write that phantom FALSE and blank the
 * navigation bar.  Both the form widget and build.Common.ThemeData
 * fall back to that one declared default, and these tests hold the
 * two of them to it.
 ******************************************/

// minimalNavigationItem stands in for a row of build.Common.Navigation: the
// top-level Streams that the theme lists in its navigation bar.
type minimalNavigationItem struct {
	StreamID string
	Token    string
	Label    string
	Icon     string
}

// minimalNavigationBuilder stands in for the build.Builder that renders the
// theme's page chrome, exposing only the fields navigation.html reads.
type minimalNavigationBuilder struct {
	DomainStateID             string
	Navigation                []minimalNavigationItem
	IsAuthenticated           bool
	IsIdentity                bool
	IsOwner                   bool
	HasUnreadNotifications    bool
	DomainHasRegistrationForm bool
	themeData                 mapof.Any
	themeSchema               schema.Schema
}

// NotAuthenticatedOrIdentity mirrors build.Common.NotAuthenticatedOrIdentity: TRUE for a
// visitor who is neither a signed-in user nor a guest identity.
func (builder minimalNavigationBuilder) NotAuthenticatedOrIdentity() bool {
	return !builder.IsAuthenticated && !builder.IsIdentity
}

// ThemeData mirrors build.Common.ThemeData: the stored value coerced to a string, falling
// back to the Theme schema's declared default for a token the Domain has never saved.
func (builder minimalNavigationBuilder) ThemeData(token string) string {

	if value, exists := builder.themeData[token]; exists {
		return convert.String(value)
	}

	element, exists := builder.themeSchema.GetElement("themeData." + token)

	if !exists {
		return ""
	}

	return convert.String(element.DefaultValue())
}

// loadMinimalTheme parses the real theme-minimal definition the same way service.Theme.Add does
func loadMinimalTheme(t *testing.T) model.Theme {

	t.Helper()

	filesystem := os.DirFS("../_embed/templates/theme-minimal")
	definitionType, definition := findDefinition(filesystem)
	require.Equal(t, DefinitionTheme, definitionType)

	theme := model.NewTheme("minimal", emissarytemplates.FuncMap(nullIconProvider{}))
	require.NoError(t, hjson.Unmarshal(definition, &theme))
	require.NoError(t, theme.Schema.ValidateFormats())

	return theme
}

// minimalDomainSchema rebuilds the merged Domain schema exactly as build.Domain.schema() does,
// so that the theme's themeData properties layer onto the Domain's own wildcard object.
func minimalDomainSchema(t *testing.T) schema.Schema {

	t.Helper()

	result := schema.New(model.DomainSchema())
	result.Inherit(loadMinimalTheme(t).Schema)
	return result
}

// renderMinimalNavigation parses the real theme-minimal folder the way the server does at
// startup, then executes its navigation template against the provided context.
func renderMinimalNavigation(t *testing.T, context minimalNavigationBuilder) string {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	themeTemplate := template.New("theme-minimal").Funcs(funcMap)

	filesystem := os.DirFS("../_embed/templates/theme-minimal")
	require.NoError(t, loadHTMLTemplateFromFilesystem(filesystem, themeTemplate, funcMap))

	var buffer strings.Builder
	require.NoError(t, themeTemplate.ExecuteTemplate(&buffer, "navigation", context))

	return buffer.String()
}

// newMinimalNavigationBuilder returns an anonymous visitor on a LIVE domain that publishes
// two top-level pages -- the shape in which the navigation bar is visible by default.
func newMinimalNavigationBuilder(t *testing.T, themeData mapof.Any) minimalNavigationBuilder {

	t.Helper()

	return minimalNavigationBuilder{
		DomainStateID: model.DomainStateLive,
		themeData:     themeData,
		themeSchema:   loadMinimalTheme(t).Schema,
		Navigation: []minimalNavigationItem{
			{StreamID: "6a908f1d331aab271b96e5f8", Token: "about", Label: "About Me"},
			{StreamID: "6a9484676097f8f87ef9a455", Token: "contact", Label: "Contact Me"},
		},
	}
}

// TestMinimalTheme_NavigationToggle_Default verifies that a Domain whose owner has never
// touched the toggle still gets its navigation bar.  This is the pristine case: ThemeData
// is empty, so the flag resolves through the Theme schema's declared default.
func TestMinimalTheme_NavigationToggle_Default(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)
	require.Contains(t, result, `href="/signin"`)
	require.Contains(t, result, `href="/register"`)
}

// TestMinimalTheme_NavigationToggle_Visible verifies the toggle in its ON position, which is
// also what the form posts for every save that leaves the toggle alone.
func TestMinimalTheme_NavigationToggle_Visible(t *testing.T) {

	result := renderMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.Any{"showNavigation": true}))

	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)
}

// TestMinimalTheme_NavigationToggle_Hidden verifies that switching the toggle OFF drops the
// whole navigation bar.  The flag wraps the entire <nav>, not just the top-level links, so
// the guest Sign In block goes with it.
func TestMinimalTheme_NavigationToggle_Hidden(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.Any{"showNavigation": false})
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.NotContains(t, result, "<nav>")
	require.NotContains(t, result, `href="/about"`)
	require.NotContains(t, result, `href="/signin"`)
}

// TestMinimalTheme_SignupToggle_Hidden verifies that the Sign In toggle drops the guest
// call-to-action WITHOUT taking the rest of the navigation bar with it.  The two flags gate
// different amounts of the bar, so they need to be pinned apart.
func TestMinimalTheme_SignupToggle_Hidden(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.Any{"showSignup": false})
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/about"`)
	require.NotContains(t, result, `href="/signin"`)
	require.NotContains(t, result, `href="/register"`)
}

// TestMinimalTheme_NavigationToggle_Startup verifies that a domain still in STARTUP reaches
// its Startup Checklist, which is the owner's path out of that state.  A domain in STARTUP
// has never saved its settings, so the flag resolves through the default here.
func TestMinimalTheme_NavigationToggle_Startup(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainStateID = model.DomainStateStartup

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/startup"`)
}

// TestMinimalTheme_SignedInShortcuts verifies that the floating menu renders outside <nav>,
// and keeps doing so when the navigation bar itself is switched off -- the two are
// independent, and a user who hides the bar must not lose their own links.
func TestMinimalTheme_SignedInShortcuts(t *testing.T) {

	for _, themeData := range []mapof.Any{
		mapof.NewAny(),
		{"showNavigation": false},
	} {
		context := newMinimalNavigationBuilder(t, themeData)
		context.IsAuthenticated = true

		result := renderMinimalNavigation(t, context)

		require.Contains(t, result, `class="floating-menu card padding-xs"`)
		require.Contains(t, result, `href="/@me"`)
		require.Contains(t, result, `href="/@me/notifications/replies"`)
	}
}

// TestMinimalTheme_FloatingMenuClassIsShared verifies that BOTH signed-in branches carry the
// same self-contained class.  The guest-identity branch used to put the positioning and the
// card on one element while the authenticated branch nested them, so a selector written for
// one shape silently missed the other -- and a missed .floating-menu is a collapsed menu.
func TestMinimalTheme_FloatingMenuClassIsShared(t *testing.T) {

	for name, setup := range map[string]func(*minimalNavigationBuilder){
		"authenticated": func(b *minimalNavigationBuilder) { b.IsAuthenticated = true },
		"identity":      func(b *minimalNavigationBuilder) { b.IsIdentity = true },
	} {
		context := newMinimalNavigationBuilder(t, mapof.NewAny())
		setup(&context)

		result := renderMinimalNavigation(t, context)

		require.Contains(t, result, `class="floating-menu card padding-xs"`, name)

		// The old two-element shape must not come back: .floating-menu carries its own
		// positioning, so pairing it with the positioning utility would double up.
		require.NotContains(t, result, `pos-absolute-bottom-right`, name)
	}
}

// TestMinimalTheme_Form_Toggles verifies that both toggles reach the rendered settings form,
// bound to the themeData properties the theme's schema declares.  A form field naming a
// property that the schema does not define renders, but silently drops its value on save.
func TestMinimalTheme_Form_Toggles(t *testing.T) {

	widget.UseAll()

	domainSchema := minimalDomainSchema(t)

	for _, path := range []string{"themeData.showNavigation", "themeData.showSignup"} {

		element, exists := domainSchema.GetElement(path)
		require.True(t, exists, "schema does not declare %s", path)
		require.IsType(t, schema.Boolean{}, element, "%s must be a boolean", path)
	}

	domain := model.NewDomain()
	settingsForm := form.Form{Schema: domainSchema, Element: loadMinimalTheme(t).Form}

	result, err := settingsForm.Editor(&domain, nil)
	require.NoError(t, err)

	require.Contains(t, result, `name="themeData.showNavigation"`)
	require.Contains(t, result, `name="themeData.showSignup"`)

	// Both toggles render ON for a Domain that has never saved them.  The toggle posts
	// whatever it displays, and the tab layout posts every field on every save, so a toggle
	// that displayed OFF here would write FALSE the first time the owner saved any tab.
	require.Equal(t, 2, strings.Count(result, `value="true"`))
}

// TestMinimalTheme_Form_SavePreservesNavigation is the polarity guard.  The settings form is a
// tab layout, so saving ANY tab posts EVERY field, including both toggles.  Saving a pristine
// Domain must therefore leave the navigation visible -- if the flags were ever renamed to
// "show" polarity, this is the test that would catch the blanked navigation bar.
func TestMinimalTheme_Form_SavePreservesNavigation(t *testing.T) {

	widget.UseAll()

	domain := model.NewDomain()
	settingsForm := form.Form{Schema: minimalDomainSchema(t), Element: loadMinimalTheme(t).Form}

	// A toggle posts whatever it displays, and both of these display ON, so this is what
	// the browser sends for a save that leaves them alone
	values := url.Values{
		"label":                    []string{"My Site"},
		"themeData.showNavigation": []string{"true"},
		"themeData.showSignup":     []string{"true"},
	}

	require.NoError(t, settingsForm.SetURLValues(&domain, values, nil))

	// It round-trips as a real boolean, not the string "true"
	require.Equal(t, true, domain.ThemeData["showNavigation"])

	result := renderMinimalNavigation(t, newMinimalNavigationBuilder(t, domain.ThemeData))
	require.Contains(t, result, `href="/about"`)

	// And switching it off still works -- the default must not override a stored FALSE
	values.Set("themeData.showNavigation", "false")
	require.NoError(t, settingsForm.SetURLValues(&domain, values, nil))
	require.Equal(t, false, domain.ThemeData["showNavigation"])

	result = renderMinimalNavigation(t, newMinimalNavigationBuilder(t, domain.ThemeData))
	require.NotContains(t, result, `href="/about"`)
}
